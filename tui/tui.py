from textual.app import App, ComposeResult
from textual.containers import Horizontal, Vertical
from textual.widgets import Button, TextLog

class TwoPaneApp(App):
    CSS = """
    Screen {
        layout: horizontal;
    }
    .pane {
        border: solid white;
        width: 1fr;
        padding: 1;
    }
    """

    def compose(self) -> ComposeResult:
        with Horizontal():
            # Left pane with buttons
            with Vertical(classes="pane"):
                yield Button("Start", id="start")
                yield Button("End", id="end")
            
            # Right pane with TextLog
            yield TextLog(classes="pane", id="log", highlight=True, markup=True)

    def on_button_pressed(self, event: Button.Pressed) -> None:
        log = self.query_one("#log", TextLog)
        if event.button.id == "start":
            log.write("[green]Start button pressed![/green]")
        elif event.button.id == "end":
            log.write("[red]End button pressed![/red]")


if __name__ == "__main__":
    TwoPaneApp().run()

