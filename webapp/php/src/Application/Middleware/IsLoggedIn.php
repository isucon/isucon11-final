<?php

declare(strict_types=1);

namespace App\Application\Middleware;

use Fig\Http\Message\StatusCodeInterface;
use Psr\Http\Message\ResponseFactoryInterface;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Psr\Http\Server\RequestHandlerInterface as RequestHandler;
use SlimSession\Helper as SessionHelper;

final class IsLoggedIn
{
    public function __construct(
        private SessionHelper $session,
        private ResponseFactoryInterface $responseFactory,
    ) {
    }

    /**
     * isLoggedIn ログイン確認用middleware
     */
    public function __invoke(Request $request, RequestHandler $handler): Response
    {
        if (!$this->session->exists('userID')) {
            $response = $this->responseFactory->createResponse();
            $response->getBody()->write('You are not logged in.');

            return $response->withStatus(StatusCodeInterface::STATUS_UNAUTHORIZED);
        }

        return $handler->handle($request);
    }
}
