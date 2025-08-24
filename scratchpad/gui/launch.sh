export PATH="/opt/X11/bin:$PATH"
#xhost +x
docker run -it --net=host --env="DISPLAY" --volume="$HOME/.Xauthority:/root/.Xauthority:rw" app /bin/sh
