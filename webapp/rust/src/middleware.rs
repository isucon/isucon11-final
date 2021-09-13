#![allow(clippy::type_complexity)]

use futures::future;

// ログイン確認用middleware
pub struct IsLoggedIn;
impl<S, B> actix_web::dev::Transform<S, actix_web::dev::ServiceRequest> for IsLoggedIn
where
    S: actix_web::dev::Service<
        actix_web::dev::ServiceRequest,
        Response = actix_web::dev::ServiceResponse<B>,
        Error = actix_web::error::Error,
    >,
{
    type Response = actix_web::dev::ServiceResponse<B>;
    type Error = actix_web::Error;
    type Transform = IsLoggedInMiddleware<S>;
    type InitError = ();
    type Future = future::Ready<Result<Self::Transform, Self::InitError>>;

    fn new_transform(&self, service: S) -> Self::Future {
        future::ok(IsLoggedInMiddleware { service })
    }
}
pub struct IsLoggedInMiddleware<S> {
    service: S,
}
impl<S, B> actix_web::dev::Service<actix_web::dev::ServiceRequest> for IsLoggedInMiddleware<S>
where
    S: actix_web::dev::Service<
        actix_web::dev::ServiceRequest,
        Response = actix_web::dev::ServiceResponse<B>,
        Error = actix_web::error::Error,
    >,
{
    type Response = actix_web::dev::ServiceResponse<B>;
    type Error = actix_web::Error;
    type Future = future::Either<S::Future, future::Ready<Result<Self::Response, Self::Error>>>;

    actix_web::dev::forward_ready!(service);

    fn call(&self, req: actix_web::dev::ServiceRequest) -> Self::Future {
        use actix_session::UserSession as _;
        use futures::FutureExt as _;

        match req.get_session().get::<String>("userID") {
            Ok(Some(_)) => self.service.call(req).left_future(),
            Ok(None) => future::err(actix_web::error::ErrorUnauthorized(
                "You are not logged in.",
            ))
            .right_future(),
            Err(e) => future::err(e).right_future(),
        }
    }
}

// admin確認用middleware
pub struct IsAdmin;
impl<S, B> actix_web::dev::Transform<S, actix_web::dev::ServiceRequest> for IsAdmin
where
    S: actix_web::dev::Service<
        actix_web::dev::ServiceRequest,
        Response = actix_web::dev::ServiceResponse<B>,
        Error = actix_web::error::Error,
    >,
{
    type Response = actix_web::dev::ServiceResponse<B>;
    type Error = actix_web::Error;
    type Transform = IsAdminMiddleware<S>;
    type InitError = ();
    type Future = future::Ready<Result<Self::Transform, Self::InitError>>;

    fn new_transform(&self, service: S) -> Self::Future {
        future::ok(IsAdminMiddleware { service })
    }
}
pub struct IsAdminMiddleware<S> {
    service: S,
}
impl<S, B> actix_web::dev::Service<actix_web::dev::ServiceRequest> for IsAdminMiddleware<S>
where
    S: actix_web::dev::Service<
        actix_web::dev::ServiceRequest,
        Response = actix_web::dev::ServiceResponse<B>,
        Error = actix_web::error::Error,
    >,
{
    type Response = actix_web::dev::ServiceResponse<B>;
    type Error = actix_web::Error;
    type Future = future::Either<S::Future, future::Ready<Result<Self::Response, Self::Error>>>;

    actix_web::dev::forward_ready!(service);

    fn call(&self, req: actix_web::dev::ServiceRequest) -> Self::Future {
        use actix_session::UserSession as _;
        use futures::FutureExt as _;

        match req.get_session().get::<bool>("isAdmin") {
            Ok(Some(true)) => self.service.call(req).left_future(),
            Ok(Some(false)) | Ok(None) => {
                future::err(actix_web::error::ErrorForbidden("You are not admin user."))
                    .right_future()
            }
            Err(e) => future::err(e).right_future(),
        }
    }
}
