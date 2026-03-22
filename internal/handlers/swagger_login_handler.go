package handlers

import (
	"html/template"
	"net/http"
)

type SwaggerLoginHandler struct {
	User string
	Pass string
}

func NewSwaggerLoginHandler(user, pass string) *SwaggerLoginHandler {
	return &SwaggerLoginHandler{
		User: user,
		Pass: pass,
	}
}

func (h *SwaggerLoginHandler) Handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.LoginPage(w, r)
	case http.MethodPost:
		h.LoginSubmit(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *SwaggerLoginHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	const tpl = `
<!DOCTYPE html>
<html lang="es">
<head>
	<meta charset="UTF-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1.0" />
	<title>Acceso documentación API</title>
	<style>
		:root {
			--bg1: #c084fc;
			--bg2: #8b5cf6;
			--bg3: #6366f1;
			--panel-left-1: #6d28d9;
			--panel-left-2: #4338ca;
			--panel-right: #f8fafc;
			--text-main: #0f172a;
			--text-soft: #64748b;
			--input-bg: #e2e8f0;
			--input-border: #cbd5e1;
			--button-1: #a855f7;
			--button-2: #8b5cf6;
			--error-bg: #fef2f2;
			--error-border: #fecaca;
			--error-text: #b91c1c;
		}

		* {
			box-sizing: border-box;
		}

		html, body {
			height: 100%;
			margin: 0;
			font-family: Inter, Arial, sans-serif;
		}

		body {
			min-height: 100vh;
			display: flex;
			align-items: center;
			justify-content: center;
			padding: 24px;
			background:
				radial-gradient(circle at top left, rgba(255,255,255,0.18), transparent 30%),
				radial-gradient(circle at bottom right, rgba(255,255,255,0.12), transparent 30%),
				linear-gradient(135deg, var(--bg1) 0%, var(--bg2) 50%, var(--bg3) 100%);
		}

		.wrapper {
			width: 100%;
			max-width: 1020px;
			min-height: 560px;
			display: grid;
			grid-template-columns: 1fr 1fr;
			border-radius: 28px;
			overflow: hidden;
			box-shadow: 0 30px 80px rgba(15, 23, 42, 0.22);
			background: white;
		}

		.left-panel {
			position: relative;
			padding: 56px 48px;
			color: white;
			background:
				radial-gradient(circle at top left, rgba(255,255,255,0.14), transparent 25%),
				linear-gradient(160deg, var(--panel-left-1) 0%, var(--panel-left-2) 100%);
			display: flex;
			flex-direction: column;
			justify-content: center;
			overflow: hidden;
		}

		.left-panel::before,
		.left-panel::after {
			content: "";
			position: absolute;
			left: -10%;
			width: 120%;
			height: 120px;
			border: 1px solid rgba(255,255,255,0.12);
			border-radius: 999px;
		}

		.left-panel::before {
			top: 38%;
			transform: rotate(6deg);
		}

		.left-panel::after {
			top: 48%;
			transform: rotate(-5deg);
		}

		.decor-dots {
			position: absolute;
			top: 40px;
			right: 40px;
			display: grid;
			grid-template-columns: repeat(6, 8px);
			gap: 6px;
			opacity: 0.45;
		}

		.decor-dots span {
			width: 8px;
			height: 8px;
			border-radius: 50%;
			background: rgba(255,255,255,0.75);
		}

		.decor-cross {
			position: absolute;
			color: rgba(255,255,255,0.7);
			font-size: 28px;
			font-weight: 300;
		}

		.decor-cross.top {
			top: 42px;
			left: 48px;
		}

		.decor-cross.bottom {
			bottom: 48px;
			left: 64px;
		}

		.left-badge {
			position: relative;
			z-index: 2;
			display: inline-flex;
			align-items: center;
			gap: 10px;
			padding: 10px 16px;
			width: fit-content;
			border-radius: 999px;
			background: rgba(255,255,255,0.12);
			border: 1px solid rgba(255,255,255,0.16);
			font-size: 13px;
			font-weight: 600;
			letter-spacing: 0.2px;
			margin-bottom: 28px;
		}

		.left-title {
			position: relative;
			z-index: 2;
			font-size: 44px;
			line-height: 1.08;
			font-weight: 800;
			margin: 0 0 18px 0;
			max-width: 420px;
		}

		.left-text {
			position: relative;
			z-index: 2;
			font-size: 22px;
			line-height: 1.55;
			color: rgba(255,255,255,0.88);
			max-width: 430px;
			margin: 0;
		}

		.right-panel {
			background: var(--panel-right);
			display: flex;
			align-items: center;
			justify-content: center;
			padding: 44px;
		}

		.form-box {
			width: 100%;
			max-width: 380px;
		}

		.form-title {
			font-size: 42px;
			font-weight: 800;
			color: var(--text-main);
			margin: 0 0 12px 0;
			text-align: center;
		}

		.form-subtitle {
			font-size: 20px;
			color: var(--text-soft);
			text-align: center;
			margin: 0 0 34px 0;
		}

		.error {
			margin-bottom: 18px;
			padding: 14px 16px;
			border-radius: 14px;
			background: var(--error-bg);
			color: var(--error-text);
			border: 1px solid var(--error-border);
			font-size: 15px;
			font-weight: 600;
		}

		.field {
			margin-bottom: 18px;
		}

		.label {
			display: block;
			margin-bottom: 8px;
			font-size: 14px;
			font-weight: 700;
			color: #334155;
		}

		.input-wrap {
			position: relative;
		}

		.input-icon {
			position: absolute;
			left: 16px;
			top: 50%;
			transform: translateY(-50%);
			width: 18px;
			height: 18px;
			color: #94a3b8;
			pointer-events: none;
		}

		.input {
			width: 100%;
			height: 58px;
			border: 1px solid var(--input-border);
			background: var(--input-bg);
			border-radius: 16px;
			padding: 0 18px 0 48px;
			font-size: 16px;
			color: var(--text-main);
			outline: none;
			transition: 0.2s ease;
			box-shadow: 0 6px 16px rgba(15, 23, 42, 0.04);
		}

		.input:focus {
			border-color: #8b5cf6;
			background: #eef2ff;
			box-shadow: 0 0 0 4px rgba(139, 92, 246, 0.15);
		}

		.meta-row {
			display: flex;
			align-items: center;
			justify-content: space-between;
			margin: 8px 0 24px 0;
			gap: 12px;
		}

		.remember {
			display: inline-flex;
			align-items: center;
			gap: 10px;
			font-size: 14px;
			color: #64748b;
			user-select: none;
		}

		.remember input {
			width: 16px;
			height: 16px;
			accent-color: #10b981;
		}

		.meta-link {
			font-size: 14px;
			color: #7c3aed;
			text-decoration: none;
			font-weight: 600;
		}

		.meta-link:hover {
			text-decoration: underline;
		}

		.submit-btn {
			width: 100%;
			height: 58px;
			border: none;
			border-radius: 16px;
			background: linear-gradient(90deg, var(--button-1) 0%, var(--button-2) 100%);
			color: white;
			font-size: 17px;
			font-weight: 800;
			cursor: pointer;
			box-shadow: 0 14px 30px rgba(139, 92, 246, 0.28);
			transition: transform 0.15s ease, box-shadow 0.15s ease;
		}

		.submit-btn:hover {
			transform: translateY(-1px);
			box-shadow: 0 18px 34px rgba(139, 92, 246, 0.32);
		}

		.footer {
			margin-top: 28px;
			display: flex;
			align-items: center;
			justify-content: space-between;
			gap: 16px;
			font-size: 13px;
			color: #94a3b8;
		}

		@media (max-width: 900px) {
			.wrapper {
				grid-template-columns: 1fr;
				max-width: 560px;
			}

			.left-panel {
				min-height: 260px;
				padding: 36px 32px;
			}

			.left-title {
				font-size: 34px;
			}

			.left-text {
				font-size: 18px;
			}

			.right-panel {
				padding: 32px 24px;
			}
		}

		@media (max-width: 520px) {
			body {
				padding: 14px;
			}

			.wrapper {
				border-radius: 22px;
			}

			.left-panel,
			.right-panel {
				padding: 24px 20px;
			}

			.form-title {
				font-size: 32px;
			}

			.form-subtitle {
				font-size: 16px;
			}

			.meta-row {
				flex-direction: column;
				align-items: flex-start;
			}

			.footer {
				flex-direction: column;
				align-items: flex-start;
			}
		}
	</style>
</head>
<body>
	<div class="wrapper">
		<div class="left-panel">
			<div class="decor-cross top">+</div>

			<div class="decor-dots">
				<span></span><span></span><span></span><span></span><span></span><span></span>
				<span></span><span></span><span></span><span></span><span></span><span></span>
				<span></span><span></span><span></span><span></span><span></span><span></span>
			</div>

			<div class="left-badge">Reloj Control · API Docs</div>

			<h1 class="left-title">Documentación técnica y acceso seguro</h1>
			<p class="left-text">
				Consulta y prueba los endpoints del sistema de control de asistencia,
				turnos, sucursales, accesos y dispositivos desde un entorno protegido.
			</p>

			<div class="decor-cross bottom">+</div>
		</div>

		<div class="right-panel">
			<div class="form-box">
				<h2 class="form-title">Iniciar sesión</h2>
				<p class="form-subtitle">Ingresa con tus credenciales</p>

				{{if .Error}}
					<div class="error">{{.Error}}</div>
				{{end}}

				<form method="POST" action="/swagger-login">
					<div class="field">
						<label class="label" for="username">Usuario</label>
						<div class="input-wrap">
							<svg class="input-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M20 21a8 8 0 0 0-16 0"></path>
								<circle cx="12" cy="7" r="4"></circle>
							</svg>
							<input class="input" id="username" name="username" type="text" autocomplete="username" required />
						</div>
					</div>

					<div class="field">
						<label class="label" for="password">Contraseña</label>
						<div class="input-wrap">
							<svg class="input-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<rect x="3" y="11" width="18" height="10" rx="2"></rect>
								<path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
							</svg>
							<input class="input" id="password" name="password" type="password" autocomplete="current-password" required />
						</div>
					</div>

					<div class="meta-row">
						<label class="remember">
							<input type="checkbox" checked disabled />
							<span>Acceso protegido</span>
						</label>

						<a class="meta-link" href="/docs" onclick="return false;">Documentación interna</a>
					</div>

					<button class="submit-btn" type="submit">Ingresar</button>
				</form>

				<div class="footer">
					<span>© 2026 Reloj Control</span>
					<span>Documentación API v1.0</span>
				</div>
			</div>
		</div>
	</div>
</body>
</html>
`

	t := template.Must(template.New("swagger-login").Parse(tpl))
	_ = t.Execute(w, map[string]any{
		"Error": r.URL.Query().Get("error"),
	})
}

func (h *SwaggerLoginHandler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/swagger-login?error=Formulario+inválido", http.StatusFound)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username != h.User || password != h.Pass {
		http.Redirect(w, r, "/swagger-login?error=Credenciales+inválidas", http.StatusFound)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "swagger_session",
		Value:    "authenticated",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   60 * 60 * 8,
		// Secure: true, // activar en https
	})

	http.Redirect(w, r, "/docs/index.html", http.StatusFound)
}

func (h *SwaggerLoginHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "swagger_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/swagger-login", http.StatusFound)
}
