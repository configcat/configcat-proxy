FROM node:14-alpine as builder

WORKDIR /configcat

COPY package*.json /configcat/

RUN npm ci
COPY . /configcat/

RUN npm run build
RUN npm prune --production

FROM node:14-alpine

ENV NODE_ENV production
ENV PORT 4224

WORKDIR /
COPY --from=builder /configcat/dist /configcat
USER node
HEALTHCHECK --interval=60s --timeout=10s --start-period=60s \  
    CMD wget -q --method=GET http://localhost:4224/health
CMD node configcat