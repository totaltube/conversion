FROM sersh/ffmegmagick

COPY totaltube-conversion /bin/
COPY .bashrc /root/
WORKDIR /data
VOLUME /data
ENTRYPOINT ["totaltube-conversion"]
EXPOSE 8080