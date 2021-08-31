FROM php:8.0.9

WORKDIR /webapp/php

RUN apt-get update && \
    apt-get install -y wget libzip-dev unzip default-mysql-client && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

RUN docker-php-ext-configure zip && \
    docker-php-ext-install zip && \
    docker-php-ext-install pdo_mysql

ENV DOCKERIZE_VERSION v0.6.1
RUN wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz

RUN ln -sf /usr/share/zoneinfo/Asia/Tokyo /etc/localtime

COPY ./php.ini /usr/local/etc/php/php.ini
COPY . .

RUN ./composer.phar install

EXPOSE 7000

ENTRYPOINT ["dockerize", "-timeout", "60s", "-wait", "tcp://mysql:3306"]
CMD ["./composer.phar", "start"]
