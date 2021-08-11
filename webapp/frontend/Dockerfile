FROM node:14 as builder
WORKDIR /src/app

COPY package*.json .
RUN npm ci

COPY . .
RUN npm run build

FROM nginx:1.21 as runner
COPY --from=builder /src/app/dist /www/data
