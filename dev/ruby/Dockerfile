# vim: ft=dockerfile
FROM public.ecr.aws/ubuntu/ubuntu:20.04
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install --no-install-recommends --no-install-suggests -y curl ca-certificates

# 本番では xbuild で 3.0.2 を入れるのでだいじょうぶです
RUN mkdir -p /usr/local/share/keyrings
RUN curl -LSsfo /usr/local/share/keyrings/sorah-ruby.asc https://sorah.jp/packaging/debian/3F0F56A8.pub.txt
RUN echo "deb [signed-by=/usr/local/share/keyrings/sorah-ruby.asc] http://cache.ruby-lang.org/lab/sorah/deb/ focal main" > /etc/apt/sources.list.d/sorah-ruby.list \
  && apt-get update \
  && apt-get upgrade -y \
  && apt-get install --no-install-recommends --no-install-suggests -y \
  ruby \
  ruby-dev \
  ruby3.0 \
  ruby3.0-dev \
  libruby3.0 \
  ruby3.0-gems \
  default-mysql-client \
  libmysqlclient-dev \
  build-essential \
  zlib1g-dev \
  tzdata \
  zip

ENV DOCKERIZE_VERSION v0.6.1
RUN curl -LSsfo /tmp/dockerize.tar.gz https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf /tmp/dockerize.tar.gz \
    && rm /tmp/dockerize.tar.gz

RUN ln -sf /usr/share/zoneinfo/Asia/Tokyo /etc/localtime

WORKDIR /webapp/ruby
COPY ./Gemfile* /webapp/ruby/
RUN bundle install --jobs 300

COPY ./ /webapp/ruby/

ENV LANG=C.UTF-8
EXPOSE 7000
CMD ["bundle", "exec", "puma", "-p", "7000"]

