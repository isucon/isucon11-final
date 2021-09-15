FROM rust:1.55.0-buster

WORKDIR /webapp/rust

RUN apt-get update && apt-get install -y zip

ARG DOCKERIZE_VERSION=v0.6.1
RUN curl -sSfLO https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz

COPY Cargo.lock /Cargo.toml ./
RUN mkdir src &&  echo 'fn main() {}' > src/main.rs && cargo build --locked && rm src/main.rs target/debug/deps/isucholar-*

COPY . ./
RUN cargo build --locked --frozen

CMD ["/webapp/rust/target/debug/isucholar"]
