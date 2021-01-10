# ==============================================================================
# Build Stage

FROM golang:1.16beta1 as build

ARG VCS_REF
ARG PKG_NAME=github.com/trezcool/masomo
ENV CGO_ENABLED=0

# Copy the source code
WORKDIR /app
COPY . .

# Build the binaries
RUN go build -o bin/ -ldflags "-X ${PKG_NAME}/core.build=${VCS_REF}" ${PKG_NAME}/apps/admin
RUN go build -o bin/ -ldflags "-X ${PKG_NAME}/core.build=${VCS_REF}" ${PKG_NAME}/apps/api


# ==============================================================================
# Final Stage

FROM alpine:3.12

# Copy binaries from build stage
WORKDIR /app
COPY --from=build /app/bin/* /app/

# Start API service
CMD ["./api"]
