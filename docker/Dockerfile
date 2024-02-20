FROM golang:1.21-alpine as build

COPY . /opt/osprober
RUN cd /opt/osprober/; \
    go build -o osprober osprober.go

RUN apk add git; \
    IMAGE_TAG=$(/opt/osprober/tools/get_version.sh); \
    echo "${IMAGE_TAG}" > /opt/osprober/image_tag.txt

FROM alpine
COPY --from=build /opt/osprober/osprober /
COPY --from=build /opt/osprober/image_tag.txt /tmp/image_tag.txt

RUN IMAGE_TAG=$(cat /tmp/image_tag.txt); \
    sh -c "echo -e \"LABELS:\n  IMAGE_TAG: ${IMAGE_TAG}\" > /dockerimage_metadata"


ENTRYPOINT ["/osprober"]
