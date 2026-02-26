package services

import (
	"context"
	"errors"
	"strings"

	"back/internal/ent"
	"back/internal/ent/accesspoint"
	"back/internal/ent/device"
)

var (
	ErrDeviceInvalidInput = errors.New("invalid device input")
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
	Direction string // "in" | "out"
	IsActive  *bool
}

type PatchDeviceInput struct {
	Name      *string
	Serial    *string
	Direction *string // "in" | "out"
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

	if accessPointID <= 0 || in.Name == "" || in.Serial == "" || (in.Direction != "in" && in.Direction != "out") {
		return nil, ErrDeviceInvalidInput
	}

	// validar que exista access_point
	if _, err := s.Client.AccessPoint.Query().Where(accesspoint.IDEQ(accessPointID)).Only(ctx); err != nil {
		return nil, err
	}

	create := s.Client.Device.Create().
		SetAccessPointID(accessPointID).
		SetName(in.Name).
		SetSerial(in.Serial).
		SetDirection(in.Direction)

	if in.IsActive != nil {
		create.SetIsActive(*in.IsActive)
	}

	return create.Save(ctx)
}

func (s *DeviceService) Patch(ctx context.Context, deviceID int, in PatchDeviceInput) (*ent.Device, error) {
	if deviceID <= 0 {
		return nil, ErrDeviceInvalidInput
	}
	if in.Name == nil && in.Serial == nil && in.Direction == nil && in.IsActive == nil {
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
		if dir != "in" && dir != "out" {
			return nil, ErrDeviceInvalidInput
		}
		upd.SetDirection(dir)
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
