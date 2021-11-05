FROM node:14-alpine as builder

WORKDIR /configcat

COPY package*.json /configcat/

RUN npm ci
COPY . /configcat/

RUN npm run build
RUN npm prune --production

FROM node:14-alpine

ENV NODE_ENV production
ENV PORT 8101

WORKDIR /
COPY --from=builder /configcat/dist /configcat
EXPOSE 8101
USER node
CMD node configcat