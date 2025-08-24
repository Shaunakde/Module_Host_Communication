# PyQt6 + X11
FROM python:3.12-slim

# Copy build context
COPY . /app/

# Set environment variable to use host's X11 display
ENV DISPLAY=:0

# System deps required for Qt/X11 rendering
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg libsm6 libxext6 \
    libgl1 libglx-mesa0 \
    libx11-6 libxext6 libxrender1 libxcb1 libxkbcommon0 libxkbcommon-x11-0 \
    libglu1-mesa xauth x11-apps \
 && rm -rf /var/lib/apt/lists/*

RUN apt-get update && apt-get install -y --no-install-recommends \
    libx11-6 libxext6 libxrender1 libxcb1 libxkbcommon0 libxkbcommon-x11-0 \
    libgl1 libglu1-mesa libegl1 libgles2 libgbm1 libglvnd0 \
    libxcb-xinerama0 libxcb-cursor0 \
    xauth x11-apps \
    freeglut3-dev \
 && rm -rf /var/lib/apt/lists/*

# Core X11/xcb, font, and GL/EGL runtime deps that Qt's xcb plugin needs
RUN apt-get update && apt-get install -y --no-install-recommends \
    # xcb / x11 basics
    libx11-6 libx11-xcb1 libxext6 libxrender1 libxrandr2 libxi6 libxfixes3 \
    libxcursor1 libxcomposite1 libxdamage1 libxxf86vm1 libxinerama1 \
    # xcb helpers (these are the typical missing ones)
    libxcb1 libxcb-render0 libxcb-shape0 libxcb-shm0 libxcb-xfixes0 \
    libxcb-icccm4 libxcb-image0 libxcb-keysyms1 libxcb-randr0 \
    libxcb-render-util0 libxcb-xinerama0 libxcb-xinput0 libxcb-xkb1 \
    libxkbcommon0 libxkbcommon-x11-0 \
    # fonts (Qt needs these for text to render)
    libfontconfig1 libfreetype6 \
    # GL / EGL (prevents libEGL.so.1 and similar errors)
    libgl1 libglu1-mesa libegl1 libgles2 libgbm1 libglvnd0 \
    # convenience tools
    xauth x11-apps \
 && rm -rf /var/lib/apt/lists/*

# PyQt6 (the wheel pulls in Qt6 binaries)
RUN pip install --no-cache-dir PyQt6

# Helpful env vars for X11 over a mounted socket
ENV QT_X11_NO_MITSHM=1 \
    QT_QPA_PLATFORM=xcb

WORKDIR /app

CMD ["python gui.py"]

# (Optional) tiny smoke test on container start; remove/change for your app
# CMD ["python", "-c", "from PyQt6.QtWidgets import QApplication,QLabel; import sys; app=QApplication(sys.argv); w=QLabel('Hello from PyQt6 in Docker'); w.resize(420,150); w.show(); app.exec()"]

