from textual.app import App, ComposeResult
from textual.containers import Horizontal, Vertical, VerticalScroll, VerticalGroup, HorizontalGroup
from textual.widgets import Button, RichLog, Log, Label, Footer, Input
from textual.binding import Binding
from textual.worker import get_current_worker
import time, datetime
import redis
import asyncio
import threading
import json
from redis.asyncio import Redis as AsyncRedis 

CHANNEL = "MODULE_Q"
class HostGUI(App):
    CSS = """
    Screen {
        layout: horizontal;
    }
    .pane {
        border: solid white;
        width: 1fr;
        padding: 1;
    }
    Input {
        margin: 1 1;
        width: 30w;
    }
    """

    BINDINGS = [
        Binding(key="q", action="quit", description="Quit the app"),
        Binding(
            key="question_mark",
            action="help",
            description="Show help screen",
            key_display="?",
        )
    ]

    # Initialize redis at the start of the class (Sync)
    r = redis.Redis(host='localhost', port=6379, db=0)

    cmd_counter = 0
    cmd_payload = {
        "CMD": "INSPECT_PANEL",
        #"CMD_PARAMS": {},
        "CMD_COUNTER": 000,
        "CMD_HASH": "23f451"
        }

    # ---------- Redis async subscriber (non-blocking) ----------
    async def _redis_subscriber(self):
        """Async Redis pub/sub loop; runs as a Textual worker."""
        self.log_widget.write(f"[green]Starting Redis subscriber to {CHANNEL}[/green]")
        r = AsyncRedis(host="localhost", port=6379, db=0) #decode_responses=True)
        pubsub = r.pubsub()
        await pubsub.subscribe("CMD_Q", "MODULE_Q") # CHANNEL)

        worker = get_current_worker()
        try:
            async for msg in pubsub.listen():
                if worker.is_cancelled:
                    break
                if msg.get("type") != "message":
                    continue
                text = msg.get("data", "")
                # Update UI (same event loop)
                
                self.log_widget.write(f"[cyan]{text}[/cyan]")
                await asyncio.sleep(0)  # yield to keep UI snappy
        finally:
            try:
                await pubsub.unsubscribe(CHANNEL)
            except Exception:
                pass
            try:
                await pubsub.close()
            except Exception:
                pass
            try:
                await r.aclose()
            except Exception:
                pass

    #async def _post_message(self, text: str):
    #    self.log_widget.write(f"[cyan]{text}[/cyan]")

# TIMER EVENTS -----------------------------------------------------
    def heartbeat(self) -> None:
        """Custom function called every 500 ms."""
        log = self.query_one("#log", RichLog)
        tstamp = time.gmtime() # USE GMT / UTC time since this is a spacecraft
        tstamp = datetime.datetime(*tstamp[:7])

        log.write(r"[cyan]HOST[/cyan] Heartbeat"+f" [white]{tstamp.isoformat()}[/white]")
        #TODO: Actually send health ping to redis list

    async def on_mount(self) -> None:
        # Cache widget refs once
        self.log_widget = self.query_one("#log", RichLog)

        # Schedule the timer: run every 0.5 seconds
        self.set_interval(0.5, self.heartbeat)

        # Start the rerdis worker
        self.sub_worker = self.run_worker(self._redis_subscriber()) #, name="redis-sub") #group="io")

    async def on_unmount(self) -> None:
        # Don't 'await' cancel(); it's not awaitable.
        if getattr(self, "sub_worker", None) is not None:
            self.sub_worker.cancel()
 
# -----------------------------------------------------------------

    def compose(self) -> ComposeResult:
        with Horizontal():
            # Left pane with buttons
            with VerticalGroup(classes="pane"):
                yield Button("INSPECT_PANEL", id="INSPECT_PANEL")
                yield Button("PERFORM_MANEUVER", id="PERFORM_MANEUVER")
                with HorizontalGroup():
                    yield Label("X")
                    yield Input(placeholder="255", type="integer")
                with HorizontalGroup():
                    yield Label("Y")
                    yield Input(placeholder="100", type="integer")
                with HorizontalGroup():
                    yield Label("Z")
                    yield Input(placeholder="155", type="integer")
                yield Button("HEALTH_CHECK", id="HEALTH_CHECK")
            # Right pane with TextLog
            with Vertical(classes="pane"):
                yield RichLog(classes="pane", id="log", markup=True, max_lines=150)
                yield Label(classes="pane", id="debug")
            yield Footer()

    def on_button_pressed(self, event: Button.Pressed) -> None:
        log = self.query_one("#log", RichLog)
        status = self.query_one("#debug", Label)
        log.write(f"[blue]Sending command:[/blue] {event.button.id}")

        if event.button.id == "INSPECT_PANEL":
            status.update("INSPECT_PANEL clicked")

            # Send a Redis Subsciption to the CMD_Q pubsub"
            json_payload = json.dumps(self.cmd_payload)
            self.r.publish("CMD_Q",json_payload)
            log.write("[green] Starting Command: Inspect Panel [/green]")

        elif event.button.id == "PERFORM_MANEUVER":
            log.write("[red]Perform Maneuver button pressed![/red]")

        elif event.button.id == "HEALTH_CHECK":
            log.write("[red]Health Check button pressed![/red]")    



if __name__ == "__main__":
    HostGUI().run()

