# Install pre-requisite
- Docker Compose

# Video Demo
Youtube: https://youtu.be/MXx2wYtvc_E

# Run Using Docker Compose (Recomended)

For convinicence and to not install the whole stack on the host machine, the application can be run using docker. Going into the following directories:

In seperate terminal tabs for each of these steps run:
- Make sure redis is running with: start_redis.sh
- gui: run "make run" to launch the gui (being in tmux helps to get the right termial environment)
- module: go to the folder and do `docker build -t module .` and `docker run -it module` to run the "backend" module software


