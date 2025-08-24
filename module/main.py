import os
import time
import json
import signal
import threading
from datetime import datetime
from typing import Optional

import redis
from proto.telemetry_pb2 import Telemetry  # generated at build

REDIS_URL   = os.getenv("REDIS_URL", "redis://localhost:6379")
STREAM_KEY  = os.getenv("STREAM_KEY", "telemetry")
ROLE        = os.getenv("ROLE", "publisher")  # "publisher" or "consumer"
GROUP_NAME  = os.getenv("GROUP_NAME", "telemetry_group")
CONSUMER    = os.getenv("CONSUMER_NAME", "worker-1")

r = redis.Redis.from_url(REDIS_URL, decode_responses=False)

_shutdown = threading.Event()

def handle_sigterm(_sig, _frm):
    _shutdown.set()

signal.signal(signal.SIGINT, handle_sigterm)
signal.signal(signal.SIGTERM, handle_sigterm)


def serialize_telemetry(i: int) -> bytes:
    """Create a demo protobuf payload."""
    msg = Telemetry(
        id=i,
        source="sensor-A",
        value=42.0 + i * 0.1,
        timestamp_ms=int(time.time() * 1000),
    )
    return msg.SerializeToString()


def publisher():
    i = 0
    print(f"[publisher] Writing Protobuf messages to Redis Stream '{STREAM_KEY}'")
    while not _shutdown.is_set():
        payload = serialize_telemetry(i)
        # Fields in Streams are key/value pairs of bytes → store protobuf in "pb" field
        r.xadd(STREAM_KEY, {"pb": payload})
        i += 1
        time.sleep(1.0)  # demo pacing
    print("[publisher] Shutting down.")


def ensure_group():
    try:
        r.xgroup_create(STREAM_KEY, GROUP_NAME, id="$", mkstream=True)
        print(f"[consumer] Created consumer group '{GROUP_NAME}' on stream '{STREAM_KEY}'")
    except redis.ResponseError as e:
        # Already exists is fine
        if "BUSYGROUP" in str(e):
            pass
        else:
            raise


def parse_and_print(entry_id: bytes, fields: dict):
    # Pull protobuf bytes from "pb"
    pb = fields.get(b"pb")
    if not pb:
        print(f"[consumer] {entry_id.decode()}: missing 'pb' field, got keys {list(fields.keys())}")
        return
    msg = Telemetry()
    msg.ParseFromString(pb)
    ts = datetime.fromtimestamp(msg.timestamp_ms / 1000.0).isoformat()
    print(f"[consumer] id={msg.id} source={msg.source} value={msg.value:.3f} ts={ts}")


def consumer():
    ensure_group()
    print(f"[consumer] Reading with group='{GROUP_NAME}', consumer='{CONSUMER}' on '{STREAM_KEY}'")
    # Block and fetch messages (new + pending) in a loop
    while not _shutdown.is_set():
        # First, claim any pending messages assigned to this consumer group (optional)
        resp = r.xreadgroup(
            groupname=GROUP_NAME,
            consumername=CONSUMER,
            streams={STREAM_KEY: '>'},  # '>' = new messages
            count=10,
            block=2000,  # ms
        )

        if not resp:
            continue

        # resp format: [(stream_key, [(entry_id, {field:value}), ...])]
        for _stream, entries in resp:
            for entry_id, fields in entries:
                try:
                    parse_and_print(entry_id, fields)
                finally:
                    # Acknowledge
                    r.xack(STREAM_KEY, GROUP_NAME, entry_id)

    print("[consumer] Shutting down.")


if __name__ == "__main__":
    if ROLE == "publisher":
        publisher()
    elif ROLE == "consumer":
        consumer()
    else:
        print(f"Unknown ROLE='{ROLE}'. Use 'publisher' or 'consumer'.")

