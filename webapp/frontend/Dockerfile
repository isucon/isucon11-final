FROM node:14 as ci
WORKDIR /src/app

COPY package*.json .
RUN npm ci

FROM nginx:1.21 as runner
COPY ./dist /www/data
