# ==============================================================================
# Build Stage

FROM golang:1.16rc1 as build

ARG MODULE
ARG VCSREF
ARG PACKAGES
ENV CGO_ENABLED=0

# Copy the source code
WORKDIR /app
COPY . .

# Build the binaries
RUN go build -o bin/ -ldflags "-X ${MODULE}/core.build=${VCSREF}" ${PACKAGES}


# ==============================================================================
# Final Stage

FROM alpine:3.12

# Copy binaries from build stage
WORKDIR /app
COPY --from=build /app/bin/* /app/

# Create app user
RUN adduser -D app
USER app

ARG VCSREF
ARG BUILD_DATE="now"
LABEL org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.title="masomo" \
      org.opencontainers.image.authors="Tresor Kambembo <kambembotresor@gmail.com>" \
      org.opencontainers.image.source="https://github.com/trezcool/masomo/backend" \
      org.opencontainers.image.revision="${VCSREF}" \
      org.opencontainers.image.vendor="Trezcool"
