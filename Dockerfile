ARG build_image=golang:1.17-bullseye
ARG base_image=debian:bullseye-slim

FROM ${build_image} AS build
RUN apt-get update && \
    apt-get install -y ca-certificates libgnutls30 make \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /build
COPY . .
RUN make build

FROM ${base_image}
COPY --from=build /build/dupe-nukem /usr/local/bin/dupe-nukem
ENTRYPOINT ["dupe-nukem"]