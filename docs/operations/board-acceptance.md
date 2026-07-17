# Trusted touch-drag board acceptance

Playwright verifies trusted mouse input and trusted touchscreen taps with the
real Chessground dependency. Its WebKit touchscreen API cannot emit a trusted
touch-drag stream, so press-drag-release behavior must also be checked on a
physical touch device. This is a temporary development preview over Tailscale,
not an application hosting requirement.

## Start the temporary tailnet preview

Prerequisites:

- install the locked frontend dependencies with `npm --prefix frontend ci`;
- connect the Mac and touch device to the same tailnet;
- allow TCP port 4173 in the applicable Tailscale ACL and macOS firewall.

From the repository root, run:

```bash
if command -v tailscale >/dev/null 2>&1; then
  TAILNET_IP="$(tailscale ip -4)"
else
  TAILNET_IP="$(/Applications/Tailscale.app/Contents/MacOS/Tailscale ip -4)"
fi

printf 'Open http://%s:4173/ on the touch device\n' "$TAILNET_IP"
npm --prefix frontend run dev -- \
  --host "$TAILNET_IP" \
  --port 4173 \
  --strictPort
```

The browser preview provides a deterministic two-puzzle session without Wails:
puzzle 1 accepts `e2e4` and treats `e2e3` as a legal-but-wrong move; puzzle 2
accepts `d2d4` and treats `d2d3` as legal-but-wrong. Complete the short learner
setup if it appears, then choose **Start today's training**.

## Physical-device checklist

Use normal finger input. Do not use browser automation, remote pointer control,
constructed JavaScript events, or accessibility click emulation for this check.

1. Tap `e2`, then tap `e3`. Confirm **Try again** appears and the pawn returns
   to `e2`.
2. Press the pawn on `e2`, drag it outside the board, and release. Confirm the
   move is cancelled, the pawn remains on `e2`, and no move feedback appears.
3. Press the pawn on `e2`, drag it to `e3`, and release. Confirm the legal but
   incorrect move snaps back and **Try again** appears.
4. Press the pawn on `e2`, drag it to `e4`, and release. Confirm the move is
   accepted.
5. Confirm the final puzzle-1 board remains visible with **Correct!**,
   **Puzzle 1 of 2**, and **Next puzzle**. Confirm it does not advance by itself,
   then choose **Next puzzle**.
6. On puzzle 2, tap `d2`, then tap `d4`. Confirm the move is accepted and the
   final board remains until **See results** is chosen.
7. Start a vertical swipe on the board. Confirm the page does not scroll while
   manipulating the board.
8. Start a vertical swipe outside the board, such as on the puzzle panel or
   page margin. Confirm ordinary page scrolling still works.

Any failure is a release blocker. Record the exact device, operating-system
version, browser/version, commit, and observation rather than retrying until a
pass is obtained.

## Evidence record

Copy this table into the implementation handoff and replace every pending cell:

| Date and timezone | Commit | Device and OS | Browser/version | Tap move | Legal drag | Cancel outside | Wrong snapback | Board scroll block | Outside scroll | Explicit Next/results | Overall |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Pending | Pending | Pending | Pending | Pending | Pending | Pending | Pending | Pending | Pending | Pending | Pending |

## Stop the preview

Press `Control-C` in the terminal that runs Vite. Confirm the temporary listener
has stopped:

```bash
lsof -nP -iTCP:4173 -sTCP:LISTEN
```

The command should print nothing and return status 1. The production Wails app
continues to require neither this preview server nor any runtime network access.
