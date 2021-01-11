# ==============================================================================
# Build Stage

FROM golang:1.16beta1 as build

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
