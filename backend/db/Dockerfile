FROM postgres:14-alpine

# Install locale dependencies
RUN apk add --no-cache icu-data-full

# Set the locale
ENV LANG=fi_FI.UTF-8
ENV LC_COLLATE=fi_FI.UTF-8
ENV LC_CTYPE=fi_FI.UTF-8
