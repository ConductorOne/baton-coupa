FROM gcr.io/distroless/static-debian11:nonroot
ENTRYPOINT ["/baton-coupa"]
COPY baton-coupa /