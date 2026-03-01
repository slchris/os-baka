FROM hub.infra.plz.ac/library/alpine:latest

RUN apk add --no-cache dnsmasq tftp-hpa curl

# Create directories
RUN mkdir -p /tftpboot /var/lib/misc /etc/dnsmasq.d

# iPXE assets: prefer build-context cache (pxe-services/ipxe), fallback to remote download if missing
WORKDIR /tftpboot
ARG IPXE_ASSET_DIR=/tmp/ipxe
COPY ipxe/ ${IPXE_ASSET_DIR}/
RUN set -e; \
        for f in undionly.kpxe ipxe.efi snponly.efi wimboot; do \
            if [ -f "${IPXE_ASSET_DIR}/$f" ]; then cp "${IPXE_ASSET_DIR}/$f" "/tftpboot/$f"; \
            else \
                url="https://boot.ipxe.org/$f"; \
                [ "$f" = "wimboot" ] && url="https://github.com/ipxe/wimboot/releases/latest/download/wimboot"; \
                echo "Downloading $f from $url"; \
                curl -fL -o "/tftpboot/$f" "$url"; \
            fi; \
        done

# Copy configuration
COPY dnsmasq.conf /etc/dnsmasq.conf
COPY boot.ipxe /tftpboot/boot.ipxe

# Ensure proper permissions
RUN chmod 644 /tftpboot/*

COPY start.sh /start.sh
RUN chmod +x /start.sh

EXPOSE 67/udp 69/udp

CMD ["/start.sh"]
