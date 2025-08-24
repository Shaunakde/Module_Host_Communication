    #!/bin/bash

    # Start a new detached tmux session
    tmux new -s my_session -d

    # Send commands to the tmux session to run your application
    # Replace `your_command_to_start_app` with the actual command for your application
    tmux send-keys -t my_session "python tui.py" C-m

    # Attach to the tmux session (optional, if you want to immediately see the app's output)
    # tmux attach -t my_session

    # Keep the container running even if tmux detaches (important for background processes)
    # This can be adjusted based on your app's needs. For a long-running app,
    # you might want to `tail -f /dev/null` or similar to keep the container alive.
    exec "$@"