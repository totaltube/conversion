FROM sersh/ffmpegmagick

COPY bin/totaltube-conversion /bin/
COPY .bashrc /root/
WORKDIR /data
VOLUME /data
ENV ISCUDA=0
COPY conversion-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/conversion-entrypoint.sh
ENTRYPOINT ["/usr/local/bin/conversion-entrypoint.sh"]
EXPOSE 8080