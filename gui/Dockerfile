FROM python:3.12-slim

# Better unicode & colors in terminals
ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    TERM=xterm-256color

WORKDIR /app
COPY . /app/

RUN apt-get update && apt-get install -y tmux
RUN tmux new -s foo -d && tmux ls

# Copy only requirements first so Docker can cache pip install layer
COPY requirements.txt .

EXPOSE 6379

RUN pip install --no-cache-dir -r requirements.txt

# Copy the rest of the source
COPY . .

#CMD ["./entrypoint.sh"]
CMD ["python", "-m", "tui"]
