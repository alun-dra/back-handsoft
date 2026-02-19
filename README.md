====================================================
BACKEND - GO
====================================================
Descripción

Backend REST desarrollado en Go.

Incluye:

Arquitectura modular

ORM Ent (PostgreSQL)

Autenticación JWT

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

Login

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
  "role": "admin",
  "username": "admin"
}
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

ESTRUCTURA DEL PROYECTO
cmd/api            → Entry point
internal/config    → Configuración
internal/database  → Conexión DB
internal/ent       → Código generado Ent
internal/services  → Lógica de negocio
internal/handlers  → Controladores HTTP
internal/middleware→ Middlewares
internal/server    → Setup del servidor
LEVANTAR EN PRODUCCIÓN

En producción:

No usar archivo .env

Configurar variables en el entorno (Railway, Docker, etc.)

Compilar:

go build -o app cmd/api/main.go
./app
NOTAS IMPORTANTES

No usar CORS "*" en producción

No exponer JWT_SECRET

Usar HTTPS siempre

No subir archivos .env al repositorio

Rotar credenciales si fueron expuestas