# syntax=docker/dockerfile:1

FROM alpine:3.20 AS build
RUN mkdir -p /out && echo gear > /out/hello.txt

FROM scratch AS result
COPY --from=build /out /
