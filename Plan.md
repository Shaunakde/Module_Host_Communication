What this demonstrates
----------------------
• A minimal message protocol (with msg_id, command, args) and idempotent handling.
• Host heartbeat @ 500 ms. Miss 3 beats ⇒ Phenix -> SAFE (latched) and rejects non-safe commands.
• Commands implemented:
- INSPECT_PANEL → OK, image_captured,<uri>
- PERFORM_MANEUVER x y z → ACCEPTED, streams PROGRESS, then RESULT (or aborts)
- HEALTH_CHECK → battery/temp result
- RESUME → exits SAFE if pre-checks pass (battery>20%, temp<60 °C)
• Host-side thrust_inhibit toggle. PERFORM_MANEUVER checks it before and during; if asserts mid-run, aborts to SAFE.
• Fault injection:
- Pause heartbeat (N beats)
- Brownout (battery ~15%) and Overtemp (temp ramps up). Overtemp throttles and can force SAFE.
• Clear ACK/RESULT tags in the log and a metrics strip (sent/accepted/rejected, SAFE entries, median ACK latency).


Message Protocol (JSON over a hypothetical link)
------------------------------------------------
Host → Phenix (examples):
{
"msg_id": "uuid-1234",
"type": "COMMAND" | "HEARTBEAT",
"cmd": "PERFORM_MANEUVER" | "INSPECT_PANEL" | "HEALTH_CHECK" | "RESUME" | "SET_THRUST_INHIBIT",
"args": { "x": 1, "y": 0, "z": -1 } // optional
}


Phenix → Host (ACKs/results):
{
"msg_id": "uuid-1234", // echoes host msg_id
"status": "ACK" | "ACCEPTED" | "REJECTED" | "PROGRESS" | "RESULT" | "ERROR",
"reason": "...", // set on REJECTED/ERROR
"data": { ... }, // e.g., battery/temp, progress, uri, etc.
"dup": true | false // true if duplicate msg_id handled idempotently
}


Timeouts & Unexpected Commands
------------------------------
• Host expects an ACK within ~1s; otherwise logs ACK TIMEOUT.
• Unknown cmd → ERROR with reason UNRECOGNIZED_COMMAND.


Extendability
-------------
• Code is organized into HostController, PhenixSimulator, and a tiny Bus layer with clear interfaces.
• Add new commands by extending PhenixSimulator.handleCommand and wiring UI buttons.
-->