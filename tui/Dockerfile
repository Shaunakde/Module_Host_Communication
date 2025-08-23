# PyQt6 + X11
FROM python:3.12-slim

# Copy build context
COPY . /app/

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
RUN pip install --no-cache-dir textual textual-dev textual[syntax]

WORKDIR /app

CMD ["python tui.py"]

