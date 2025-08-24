from textual.app import App, ComposeResult
from textual.containers import Horizontal, Vertical, VerticalScroll, VerticalGroup, HorizontalGroup
from textual.widgets import Button, RichLog, Log, Label, Footer, Input, Static, Switch
from textual.binding import Binding
from textual.worker import get_current_worker
import time, datetime
from datetime import timezone
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
    .pane_small {
        border: solid grey;
        width: 1fr;
        padding: 1;
        text-style: italic;
    }
    #InputPane {
        width: 25%;
        height: 100%;
    }
    #OutputPane {
        width: 75%;
        height: 100%;
    }
    Input {
        margin: 1 1;
        width: 40w;
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

    heartbeat_skip_count = None

    cmd_counter = 0
    cmd_payload = {
        "CMD": "INSPECT_PANEL",
        #"CMD_PARAMS": {},
        "CMD_COUNTER": 0,
        "CMD_HASH": "23f451"
        }
    
    def _create_command(self, CMD="INSPECT_PANEL"):
        cmd_payload = self.cmd_payload
        self.cmd_counter += 1
        cmd_payload["CMD_COUNTER"] = self.cmd_counter
        cmd_payload["CMD"] = CMD
        cmd_payload["CMD_HASH"] = str(hash(frozenset(cmd_payload.items())))
        return cmd_payload

    # ---------- Redis async subscriber (non-blocking) ----------
    async def _parse_redis_message(self, msg: str):
        
        _KEY_DISP = ["message", "system_state"]
        _KEY_FORMAT = ["system_state"]
        """Parse a Redis message and update the UI accordingly."""
        try:
            data = json.loads(msg)

            self.output_widget.update(json.dumps(data, indent=1))
            # Work on the parsed redis message here to show outputs?
            # There should be a key called "return_params" (list)
            # Also a key called "type" = "RET_VALUE" for output packets


            if data.get("type") == "RET_VALUE":
                # Clear return pane
                self.return_widget.clear()
                self.return_widget.write(f"{datetime.datetime.now(timezone.utc).strftime('%Y-%m-%d %H:%M:%S')} - RX")
                return_params = data.get("return_params")
                for param in return_params:
                    self.return_widget.write(f"[green]Ret: {param}[/green]") ###################

            # TODO: Thrust
            # TODO: Image URI etc.


            self.log_widget.write(f"Parsed PubSub: [blue]{data}[/blue]")
            command = data.get("message", "")
            params = data.get("system_state", {})

            for key, value in data.items():
                if key in _KEY_DISP:
                    # TODO: Think about the debug panel
                    if key in _KEY_FORMAT:
                        formatted_value = json.dumps(value, indent=2)
                        self.debug_widget.update(f"[yellow]-> {key} :\n{formatted_value}[/yellow]")
                    self.log_widget.write(f"[yellow]-> {key} : {value}[/yellow]")
            

        except json.JSONDecodeError:
            self.log_widget.write(f"[red]Failed to decode message: {msg}[/red]")

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
                self.log_widget.write(f"Raw PubSub: [cyan]{text}[/cyan]")
                await self._parse_redis_message(msg=text)
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
    def cleanup(self) -> None:
        # Trim Redis list to last 100 entries
        self.host_debug_widget.update("Cleaning up Redis list...")
        self.r.ltrim("HOST_HEARTBEAT", 0, 99)

    def heartbeat(self) -> None:
        log = self.query_one("#log", RichLog)
        tstamp = time.gmtime() # USE GMT / UTC time since this is a spacecraft
        tstamp = datetime.datetime(*tstamp[:7])

        #log.write(r"[cyan]HOST[/cyan] Heartbeat "+f" [white]{tstamp.isoformat()}[/white]")

        #TODO: Actually send health ping to redis list
        # Check on fault injection switch before sending
        skip_beat = self.query_one("#skip_beat_switch", Switch)
        num_skip_beat = self.query_one("#n_beats_input", Input).value
        if num_skip_beat != "":
            num_skip_beat = int(num_skip_beat)
        if not skip_beat.value:
            # SEND HEARTBEAT
            log.write(r"[cyan]HOST[/cyan] Heartbeat "+f" [white]{tstamp.isoformat()}[/white]")
            self.r.lpush("HOST_HEARTBEAT", f"{tstamp.isoformat()}")
        else:
            self.log_widget.write(f"[green]Skipping Heartbeat as per user request - {self.heartbeat_skip_count} [/green]")
            if self.heartbeat_skip_count is None:
                self.heartbeat_skip_count = num_skip_beat
            elif self.heartbeat_skip_count >= 1:
                self.heartbeat_skip_count -= 1
            elif self.heartbeat_skip_count <= 0:
                self.heartbeat_skip_count = None
                skip_beat.value = False

    async def on_mount(self) -> None:
        # Cache widget refs once
        self.log_widget = self.query_one("#log", RichLog)
        self.debug_widget = self.query_one("#debug", Label)
        self.output_widget = self.query_one("#output", Label)
        self.host_debug_widget = self.query_one("#host_debug", Label)
        self.return_widget = self.query_one("#return", RichLog)

        # Schedule the timer: run every 0.5 seconds
        self.set_interval(0.5, self.heartbeat)

        # Schedule the cleanup: run every 15 seconds
        self.set_interval(15, self.cleanup)

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
            with VerticalGroup(classes="pane", id="InputPane"):
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

                # Middle pane with Switch
                with VerticalGroup(classes="pane"):
                    with VerticalGroup():
                        # TODO: Implement N beats box
                        with HorizontalGroup():
                            yield Label("Skip Heartbeat:")
                            yield Switch(animate=True, value=False, id="skip_beat_switch")
                        with HorizontalGroup():
                            yield Label("N Beats:")
                            yield Input(placeholder="5", value="5", type="integer", id="n_beats_input")
 
            # Right pane with TextLog
            with Vertical(classes="pane", id="OutputPane"):
                yield RichLog(classes="pane", id="log", markup=True, max_lines=150)
                with Horizontal(classes="pane"):
                    yield Label(classes="pane", id="debug")
                    yield Label(classes="pane", id="output")
                    yield RichLog(classes="pane", id="return", markup=True, max_lines=150)
                yield Label(classes="pane_small", id="host_debug")
            yield Footer()

    def on_button_pressed(self, event: Button.Pressed) -> None:
        log = self.query_one("#log", RichLog)
        status = self.query_one("#debug", Label)
        log.write(f"[blue]Sending command:[/blue] {event.button.id}")

        if event.button.id == "INSPECT_PANEL":
            self.host_debug_widget.update("INSPECT_PANEL clicked")
            log.write("[red]Inspect Panel button pressed![/red]")

            # Send a Redis Subsciption to the CMD_Q pubsub"
            cmd = self._create_command("INSPECT_PANEL")
            json_payload = json.dumps(cmd)
            self.r.publish("CMD_Q",json_payload)
            log.write("[green] Starting Command: Inspect Panel [/green]")

        elif event.button.id == "PERFORM_MANEUVER":
            log.write("[red]Perform Maneuver button pressed![/red]")

        elif event.button.id == "HEALTH_CHECK":
            self.host_debug_widget.update("HEALTH_CHECK clicked")
            log.write("[red]Health Check button pressed![/red]")

            # Send a Redis Subsciption to the CMD_Q pubsub"
            cmd = self._create_command("HEALTH_CHECK")
            json_payload = json.dumps(cmd)
            self.r.publish("CMD_Q",json_payload)
            log.write("[green] Starting Command: Health Check [/green]")



if __name__ == "__main__":
    HostGUI().run()

