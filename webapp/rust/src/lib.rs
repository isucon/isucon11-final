use actix_web::http::StatusCode;
use actix_web::HttpResponse;

pub mod db;
pub mod middleware;
pub mod util;

#[derive(Debug, serde::Serialize)]
pub struct IsucholarError {
    #[serde(skip)]
    code: StatusCode,
    message: &'static str,
}
impl std::fmt::Display for IsucholarError {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        write!(f, "code={} message={}", self.code, self.message)
    }
}
impl actix_web::ResponseError for IsucholarError {
    fn status_code(&self) -> StatusCode {
        self.code
    }

    fn error_response(&self) -> HttpResponse {
        HttpResponse::build(self.code).json(self)
    }
}
impl IsucholarError {
    pub fn bad_request(message: &'static str) -> actix_web::Error {
        Self {
            code: StatusCode::BAD_REQUEST,
            message,
        }
        .into()
    }

    pub fn unauthorized(message: &'static str) -> actix_web::Error {
        Self {
            code: StatusCode::UNAUTHORIZED,
            message,
        }
        .into()
    }

    pub fn forbidden(message: &'static str) -> actix_web::Error {
        Self {
            code: StatusCode::FORBIDDEN,
            message,
        }
        .into()
    }

    pub fn not_found(message: &'static str) -> actix_web::Error {
        Self {
            code: StatusCode::NOT_FOUND,
            message,
        }
        .into()
    }

    pub fn conflict(message: &'static str) -> actix_web::Error {
        Self {
            code: StatusCode::CONFLICT,
            message,
        }
        .into()
    }

    pub fn internal_server_error(message: &'static str) -> actix_web::Error {
        Self {
            code: StatusCode::INTERNAL_SERVER_ERROR,
            message,
        }
        .into()
    }
}
