FROM node:14-alpine as builder

WORKDIR /configcat
RUN npm install -g ts-node
COPY package*.json /configcat/
RUN npm ci
COPY . /configcat/
RUN npm run build

FROM node:14-alpine

ENV NODE_ENV production
ENV PORT 4224

WORKDIR /configcat
COPY --from=builder /configcat/dist /configcat
COPY --from=builder /configcat/src/pre-start/env/production.env /configcat/pre-start/env/production.env
COPY package*.json /configcat/
RUN npm install --only=production
USER node

HEALTHCHECK --interval=60s --timeout=10s --start-period=60s \  
    CMD wget -q --method=GET http://localhost:4224/health
CMD node ./