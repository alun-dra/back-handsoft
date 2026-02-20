====================================================
BACKEND - GO
====================================================
Descripción

Backend REST desarrollado en Go.

Incluye:

Arquitectura modular

ORM Ent (PostgreSQL)

Autenticación JWT (Access + Refresh)

Rotación automática de refresh tokens

Máximo 3 sesiones activas por usuario

Logout individual y logout global

Middleware CORS

Middleware RequestID + Logger

Control por roles

Soporte multi-ambiente (development / production)

REQUISITOS

Go 1.22+
https://go.dev/dl/

Base de datos PostgreSQL (Railway u otra)

Archivo de variables de entorno:

.env.development

.env.production

INSTALACIÓN INICIAL

Clonar el repositorio

Entrar a la carpeta backend

cd backend

Descargar dependencias

go mod tidy
CONFIGURACIÓN DE VARIABLES DE ENTORNO

Para desarrollo local crear:

.env.development

Ejemplo:

ENV=development
PORT=8080

DATABASE_URL=postgresql://USER:PASSWORD@HOST:PORT/DB

JWT_SECRET=pon_un_secreto_largo
JWT_ISSUER=dominio-api-development
JWT_AUDIENCE=web,ios,android
JWT_ACCESS_TTL_MINUTES=60
JWT_REFRESH_TTL_DAYS=30

CORS_ALLOWED_ORIGINS=https://www.dev-dominio.cl,http://localhost:4200
CORS_ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Authorization,Content-Type,Accept
CORS_ALLOW_CREDENTIALS=false

REQUEST_TIMEOUT_SECONDS=15
LOG_LEVEL=debug
MIGRACIONES (ENT)

El proyecto usa Ent ORM.

Al iniciar la aplicación se ejecuta automáticamente:

client.Schema.Create(...)

Esto:

Crea tablas si no existen

Agrega nuevas columnas

Crea índices

En producción más adelante se recomienda usar migraciones versionadas.

LEVANTAR EN DESARROLLO
go run cmd/api/main.go

Verificar:

http://localhost:8080/health

Debe responder:

OK
AUTENTICACIÓN

Se usa JWT (HS256).

El sistema implementa:

Access Token (vida corta)

Refresh Token (vida larga)

Rotación automática

Hash seguro de refresh tokens en base de datos

Máximo 3 sesiones activas por usuario

Logout real (revocación en base de datos)

FLUJO DE AUTENTICACIÓN

Login → obtener access + refresh

Usar access token para endpoints protegidos

Cuando el access expire → usar refresh

Logout al cerrar sesión

Logout-all si se detecta actividad sospechosa

LOGIN

POST

/api/v1/auth/login

Body:

{
  "username": "admin",
  "password": "admin123"
}

Respuesta:

{
  "access_token": "...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "expires_at": "...",
  "refresh_token": "...",
  "refresh_expires_at": "...",
  "role": "admin",
  "username": "admin"
}
REFRESH TOKEN

POST

/api/v1/auth/refresh

Body:

{
  "refresh_token": "TOKEN_DEL_LOGIN"
}

Comportamiento:

Devuelve nuevo access token

Devuelve nuevo refresh token

Revoca automáticamente el anterior

Protege contra replay attacks

LOGOUT

Revoca una sesión específica.

POST

/api/v1/auth/logout

Body:

{
  "refresh_token": "TOKEN_ACTUAL"
}

Respuesta:

204 No Content
LOGOUT ALL

Revoca todas las sesiones activas del usuario.

POST

/api/v1/auth/logout-all

Header obligatorio:

Authorization: Bearer <access_token>
SESIONES ACTIVAS

Listar sesiones activas del usuario.

GET

/api/v1/auth/sessions

Header obligatorio:

Authorization: Bearer <access_token>

Respuesta ejemplo:

{
  "count": 2,
  "sessions": [
    {
      "id": 10,
      "created_at": "...",
      "expires_at": "..."
    }
  ]
}

El sistema mantiene un máximo de 3 sesiones activas por usuario.
Si se crea una cuarta sesión, la más antigua se revoca automáticamente.

RUTAS PROTEGIDAS

Para acceder a endpoints protegidos:

Header obligatorio:

Authorization: Bearer <access_token>

Ejemplo:

Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
CONTROL DE ROLES

Actualmente:

admin → puede registrar usuarios

user → acceso limitado

El endpoint:

/api/v1/auth/register

Está protegido y requiere rol admin.

REGISTER

POST

/api/v1/auth/register

Header:

Authorization: Bearer <access_token_admin>

Body:

{
  "username": "nuevo_usuario",
  "password": "123456",
  "role": "user"
}

Si no se envía role, se asigna automáticamente "user".

Errores posibles:

400 → datos inválidos

409 → usuario ya existe

401 → no autorizado

403 → rol insuficiente

TABLA RESUMEN DE ENDPOINTS
Método	Endpoint	Protegido	Descripción
POST	/auth/login	❌	Login
POST	/auth/refresh	❌	Rotar refresh
POST	/auth/logout	❌	Revocar refresh
POST	/auth/logout-all	✅	Revocar todas sesiones
POST	/auth/register	✅ (admin)	Crear usuario
GET	/auth/sessions	✅	Listar sesiones
GET	/me	✅	Usuario actual
ESTRUCTURA DEL PROYECTO
cmd/api              → Entry point
internal/config      → Configuración
internal/database    → Conexión DB
internal/ent         → Código generado Ent
internal/services    → Lógica de negocio
internal/handlers    → Controladores HTTP
internal/middleware  → Middlewares
internal/server      → Setup del servidor
LEVANTAR EN PRODUCCIÓN

En producción:

No usar archivo .env

Configurar variables en el entorno (Railway, Docker, etc.)

Usar HTTPS obligatorio

Compilar:

go build -o app cmd/api/main.go
./app
SEGURIDAD IMPLEMENTADA

JWT firmado HS256

Validación de issuer y audience

Refresh tokens hasheados en base de datos

Rotación automática anti-replay

Máximo 3 sesiones activas

Revocación individual y global

Control por roles

Transacciones para evitar condiciones de carrera

Middleware de timeout

Middleware de recuperación de pánico

NOTAS IMPORTANTES

No usar CORS "*" en producción

No exponer JWT_SECRET

Usar HTTPS siempre

No subir archivos .env al repositorio

Rotar credenciales si fueron expuestas

Supervisar sesiones activas regularmente

ESTADO ACTUAL DEL SISTEMA DE AUTENTICACIÓN

✔ Access Token
✔ Refresh Token
✔ Rotación automática
✔ Logout real
✔ Logout global
✔ Máximo 3 sesiones
✔ Hash seguro en base de datos
✔ Control por roles
✔ Arquitectura modular
✔ Multi-ambiente