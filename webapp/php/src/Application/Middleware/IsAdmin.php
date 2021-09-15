<?php

declare(strict_types=1);

namespace App\Application\Middleware;

use Fig\Http\Message\StatusCodeInterface;
use Psr\Http\Message\ResponseFactoryInterface;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Psr\Http\Server\RequestHandlerInterface as RequestHandler;
use Psr\Log\LoggerInterface;
use SlimSession\Helper as SessionHelper;

final class IsAdmin
{
    public function __construct(
        private SessionHelper $session,
        private LoggerInterface $logger,
        private ResponseFactoryInterface $responseFactory,
    ) {
    }

    /**
     * isAdmin admin確認用middleware
     */
    public function __invoke(Request $request, RequestHandler $handler): Response
    {
        if (!$this->session->exists('isAdmin')) {
            $this->logger->error('failed to get isAdmin from session');

            return $this->responseFactory
                ->createResponse()
                ->withStatus(StatusCodeInterface::STATUS_INTERNAL_SERVER_ERROR);
        }

        if (!$this->session->get('isAdmin')) {
            $response = $this->responseFactory->createResponse();
            $response->getBody()->write('You are not admin user.');

            return $response->withStatus(StatusCodeInterface::STATUS_FORBIDDEN);
        }

        return $handler->handle($request);
    }
}
