FROM nginx:1.30.3-alpine@sha256:0d3b80406a13a767339fbe2f41406d6c7da727ab89cf8fae399e81f780f814d1

ARG APP_NAME
ARG VERSION=dev

LABEL org.opencontainers.image.title="Alpha Admin" \
      org.opencontainers.image.description="Alpha administration frontend" \
      org.opencontainers.image.version="${VERSION}"

COPY docker/nginx/frontend.conf /etc/nginx/conf.d/default.conf
COPY src/frontend/apps/${APP_NAME}/dist/ /usr/share/nginx/html/
