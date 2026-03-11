package services

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"back/internal/ent"
	"back/internal/ent/accesspoint"
	"back/internal/ent/device"
)

var (
	ErrDeviceInvalidInput      = errors.New("invalid device input")
	ErrDeviceDirectionConflict = errors.New("device direction conflict")
	ErrDeviceUsernameTaken     = errors.New("device username already exists")
)

type DeviceService struct {
	Client *ent.Client
}

func NewDeviceService(client *ent.Client) *DeviceService {
	return &DeviceService{Client: client}
}

type CreateDeviceInput struct {
	Name      string
	Serial    string
	Direction string // "in" | "out" | "both"
	Username  string
	Password  string
	IsActive  *bool
}

type PatchDeviceInput struct {
	Name      *string
	Serial    *string
	Direction *string // "in" | "out" | "both"
	Username  *string
	Password  *string
	IsActive  *bool
}

func (s *DeviceService) ListForAccessPoint(ctx context.Context, accessPointID int) ([]*ent.Device, error) {
	return s.Client.Device.
		Query().
		Where(device.AccessPointIDEQ(accessPointID)).
		Order(ent.Asc(device.FieldDirection), ent.Asc(device.FieldName)).
		All(ctx)
}

func (s *DeviceService) CreateForAccessPoint(ctx context.Context, accessPointID int, in CreateDeviceInput) (*ent.Device, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Serial = strings.TrimSpace(in.Serial)
	in.Direction = strings.ToLower(strings.TrimSpace(in.Direction))
	in.Username = strings.TrimSpace(in.Username)
	in.Password = strings.TrimSpace(in.Password)

	if accessPointID <= 0 ||
		in.Name == "" ||
		in.Serial == "" ||
		in.Username == "" ||
		in.Password == "" ||
		(in.Direction != "in" && in.Direction != "out" && in.Direction != "both") {
		return nil, ErrDeviceInvalidInput
	}

	// validar que exista access_point
	if _, err := s.Client.AccessPoint.Query().Where(accesspoint.IDEQ(accessPointID)).Only(ctx); err != nil {
		return nil, err
	}

	// validar username único
	exists, err := s.Client.Device.
		Query().
		Where(device.UsernameEQ(in.Username)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDeviceUsernameTaken
	}

	if err := s.validateDirectionConflict(ctx, accessPointID, in.Direction, 0); err != nil {
		return nil, err
	}

	passwordHash, err := hashPassword(in.Password)
	if err != nil {
		return nil, err
	}

	create := s.Client.Device.Create().
		SetAccessPointID(accessPointID).
		SetName(in.Name).
		SetSerial(in.Serial).
		SetDirection(in.Direction).
		SetUsername(in.Username).
		SetPasswordHash(passwordHash).
		SetRole("device")

	if in.IsActive != nil {
		create.SetIsActive(*in.IsActive)
	}

	return create.Save(ctx)
}

func (s *DeviceService) Patch(ctx context.Context, deviceID int, in PatchDeviceInput) (*ent.Device, error) {
	if deviceID <= 0 {
		return nil, ErrDeviceInvalidInput
	}
	if in.Name == nil &&
		in.Serial == nil &&
		in.Direction == nil &&
		in.Username == nil &&
		in.Password == nil &&
		in.IsActive == nil {
		return nil, ErrDeviceInvalidInput
	}

	d, err := s.Client.Device.Get(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	upd := d.Update()

	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if n == "" {
			return nil, ErrDeviceInvalidInput
		}
		upd.SetName(n)
	}

	if in.Serial != nil {
		ser := strings.TrimSpace(*in.Serial)
		if ser == "" {
			return nil, ErrDeviceInvalidInput
		}
		upd.SetSerial(ser)
	}

	if in.Direction != nil {
		dir := strings.ToLower(strings.TrimSpace(*in.Direction))
		if dir != "in" && dir != "out" && dir != "both" {
			return nil, ErrDeviceInvalidInput
		}
		if err := s.validateDirectionConflict(ctx, d.AccessPointID, dir, d.ID); err != nil {
			return nil, err
		}
		upd.SetDirection(dir)
	}

	if in.Username != nil {
		username := strings.TrimSpace(*in.Username)
		if username == "" {
			return nil, ErrDeviceInvalidInput
		}

		exists, err := s.Client.Device.
			Query().
			Where(device.UsernameEQ(username)).
			Where(device.IDNEQ(d.ID)).
			Exist(ctx)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrDeviceUsernameTaken
		}

		upd.SetUsername(username)
	}

	if in.Password != nil {
		password := strings.TrimSpace(*in.Password)
		if password == "" {
			return nil, ErrDeviceInvalidInput
		}
		passwordHash, err := hashPassword(password)
		if err != nil {
			return nil, err
		}
		upd.SetPasswordHash(passwordHash)
	}

	if in.IsActive != nil {
		upd.SetIsActive(*in.IsActive)
	}

	return upd.Save(ctx)
}

func (s *DeviceService) Delete(ctx context.Context, deviceID int) error {
	if deviceID <= 0 {
		return ErrDeviceInvalidInput
	}
	return s.Client.Device.DeleteOneID(deviceID).Exec(ctx)
}

func hashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *DeviceService) validateDirectionConflict(ctx context.Context, accessPointID int, newDirection string, excludeDeviceID int) error {
	devicesQuery := s.Client.Device.Query().
		Where(device.AccessPointIDEQ(accessPointID))

	if excludeDeviceID > 0 {
		devicesQuery = devicesQuery.Where(device.IDNEQ(excludeDeviceID))
	}

	existing, err := devicesQuery.All(ctx)
	if err != nil {
		return err
	}

	for _, d := range existing {
		switch newDirection {
		case "both":
			// si ya existe cualquier device en ese access point, no puede entrar "both"
			return ErrDeviceDirectionConflict

		case "in":
			// no puede existir otro "in" ni uno "both"
			if d.Direction == "in" || d.Direction == "both" {
				return ErrDeviceDirectionConflict
			}

		case "out":
			// no puede existir otro "out" ni uno "both"
			if d.Direction == "out" || d.Direction == "both" {
				return ErrDeviceDirectionConflict
			}
		}
	}

	return nil
}
