from __future__ import annotations

import codecs
from dataclasses import dataclass

DEFAULT_COLS = 120
DEFAULT_ROWS = 40
DEFAULT_TAB_SIZE = 8
MAX_SCROLLBACK_LINES = 5_000


def _clamp_cols(cols: int) -> int:
    return max(4, min(500, int(cols)))


def _clamp_rows(rows: int) -> int:
    return max(4, min(500, int(rows)))


def _clamp_tab_size(tab_size: int) -> int:
    return max(1, min(16, int(tab_size)))


def _parse_numeric_param(raw: str | None, fallback: int) -> int:
    if not raw:
        return fallback
    cleaned = raw
    if cleaned and cleaned[0] in {"?", ">", "!"}:
        cleaned = cleaned[1:]
    try:
        parsed = int(cleaned)
    except ValueError:
        return fallback
    if parsed < 0:
        return fallback
    return parsed


@dataclass(frozen=True)
class TerminalFrame:
    seq: int
    text: str
    cols: int
    rows: int
    cursor_row: int
    cursor_col: int

    def to_payload(self) -> dict[str, object]:
        return {
            "type": "terminal_frame",
            "seq": self.seq,
            "screen": {
                "text": self.text,
                "cols": self.cols,
                "rows": self.rows,
                "cursor": {"row": self.cursor_row, "col": self.cursor_col},
                "format_spans": [],
            },
        }


@dataclass(frozen=True)
class TerminalUpdate:
    frame: TerminalFrame
    responses: bytes


class TerminalEmulator:
    """Incremental VT-ish text screen model used by the web terminal."""

    def __init__(
        self,
        *,
        cols: int = DEFAULT_COLS,
        tab_size: int = DEFAULT_TAB_SIZE,
        preserve_scrollback_on_clear: bool = True,
    ) -> None:
        self.cols = _clamp_cols(cols)
        self.tab_size = _clamp_tab_size(tab_size)
        self.preserve_scrollback_on_clear = preserve_scrollback_on_clear

        self.lines: list[list[str]] = [[]]
        self.row = 0
        self.col = 0
        self.saved_row = 0
        self.saved_col = 0
        self._pending = ""

    def set_cols(self, cols: int) -> None:
        self.cols = _clamp_cols(cols)
        self._clamp_cursor()

    def text(self) -> str:
        return "\n".join("".join(line) for line in self.lines)

    def feed(self, chunk: str) -> bytes:
        text = self._pending + chunk
        self._pending = ""
        responses = bytearray()

        i = 0
        while i < len(text):
            ch = text[i]
            if ch == "\x1b":
                if i + 1 >= len(text):
                    self._pending = text[i:]
                    break
                nxt = text[i + 1]

                if nxt == "[":
                    end = i + 2
                    while end < len(text):
                        value = text[end]
                        if "@" <= value <= "~":
                            break
                        end += 1
                    if end >= len(text):
                        self._pending = text[i:]
                        break
                    command = text[end]
                    params_raw = text[i + 2 : end]
                    self._apply_csi(command, params_raw, responses)
                    i = end + 1
                    continue

                if nxt == "]":
                    consumed = self._consume_osc(text[i + 2 :], responses)
                    if consumed < 0:
                        self._pending = text[i:]
                        break
                    i += 2 + consumed
                    continue

                if nxt in {"P", "_", "^"}:
                    dcs_end = text.find("\x1b\\", i + 2)
                    if dcs_end < 0:
                        self._pending = text[i:]
                        break
                    i = dcs_end + 2
                    continue

                if nxt == "7":
                    self.saved_row = self.row
                    self.saved_col = self.col
                    i += 2
                    continue

                if nxt == "8":
                    self.row = max(0, self.saved_row)
                    self.col = max(0, min(self.cols, self.saved_col))
                    self._ensure_row(self.row)
                    i += 2
                    continue

                if nxt == "D":
                    self._line_feed(carriage_return=False)
                    i += 2
                    continue

                if nxt == "E":
                    self._line_feed(carriage_return=True)
                    i += 2
                    continue

                if nxt == "M":
                    self.row = max(0, self.row - 1)
                    i += 2
                    continue

                if nxt == "c":
                    self.lines = [[]]
                    self.row = 0
                    self.col = 0
                    i += 2
                    continue

                if nxt in {"(", ")", "*", "+", "-", "."}:
                    if i + 2 < len(text):
                        i += 3
                    else:
                        i += 2
                    continue

                i += 2
                continue

            if ch == "\r":
                self.col = 0
                i += 1
                continue
            if ch == "\n":
                self._line_feed(carriage_return=True)
                i += 1
                continue
            if ch == "\b":
                self.col = max(0, self.col - 1)
                i += 1
                continue
            if ch == "\t":
                self._write_tab()
                i += 1
                continue

            code = ord(ch)
            if (0 <= code <= 8) or (11 <= code <= 12) or (14 <= code <= 31) or code == 127:
                i += 1
                continue

            self._write_char(ch)
            i += 1

        return bytes(responses)

    def _consume_osc(self, tail: str, responses: bytearray) -> int:
        # tail starts after ESC ]
        j = 0
        while j < len(tail):
            c = tail[j]
            if c == "\x07":
                payload = tail[:j]
                self._handle_osc(payload, responses)
                return j + 1
            if c == "\x1b":
                if j + 1 >= len(tail):
                    return -1
                if tail[j + 1] == "\\":
                    payload = tail[:j]
                    self._handle_osc(payload, responses)
                    return j + 2
            j += 1
        return -1

    def _handle_osc(self, payload: str, responses: bytearray) -> None:
        if payload.startswith("10;?"):
            responses.extend(b"\x1b]10;rgb:cccc/cccc/cccc\x1b\\")
            return
        if payload.startswith("11;?"):
            responses.extend(b"\x1b]11;rgb:0000/0000/0000\x1b\\")
            return

    def _trim_scrollback(self) -> None:
        if len(self.lines) <= MAX_SCROLLBACK_LINES:
            return
        drop_count = len(self.lines) - MAX_SCROLLBACK_LINES
        del self.lines[:drop_count]
        self.row = max(0, self.row - drop_count)
        self.saved_row = max(0, self.saved_row - drop_count)

    def _ensure_row(self, row: int) -> None:
        while len(self.lines) <= row:
            self.lines.append([])

    def _clamp_cursor(self) -> None:
        if self.row < 0:
            self.row = 0
        self._ensure_row(self.row)
        if self.col < 0:
            self.col = 0
        if self.col > self.cols:
            self.col = self.cols

    def _ensure_col(self, target_col: int | None = None) -> None:
        if target_col is None:
            target_col = self.col
        line = self.lines[self.row]
        while len(line) < target_col:
            line.append(" ")

    def _line_feed(self, *, carriage_return: bool) -> None:
        self.row += 1
        if carriage_return:
            self.col = 0
        self._ensure_row(self.row)
        self._trim_scrollback()
        self._clamp_cursor()

    def _write_char(self, value: str) -> None:
        if self.col >= self.cols:
            self._line_feed(carriage_return=True)
        self._ensure_col(self.col)
        line = self.lines[self.row]
        if self.col < len(line):
            line[self.col] = value
        else:
            line.append(value)
        self.col += 1
        if self.col >= self.cols:
            self._line_feed(carriage_return=True)

    def _write_tab(self) -> None:
        spaces = self.tab_size - (self.col % self.tab_size or 0)
        for _ in range(spaces):
            self._write_char(" ")

    def _clear_current_line(self) -> None:
        self.lines[self.row] = []
        self.col = 0

    def _clear_line_from_cursor(self) -> None:
        line = self.lines[self.row]
        self.lines[self.row] = line[: max(0, self.col)]

    def _clear_line_to_cursor(self) -> None:
        self._ensure_col(self.col)
        line = self.lines[self.row]
        for idx in range(0, self.col):
            line[idx] = " "

    def _clear_display(self, mode: int) -> None:
        if mode == 3:
            self.lines = [[]]
            self.row = 0
            self.col = 0
            return

        if mode == 2:
            if self.preserve_scrollback_on_clear:
                self.row = len(self.lines)
                self.col = 0
                self.lines.append([])
                self._trim_scrollback()
                return
            self.lines = [[]]
            self.row = 0
            self.col = 0
            return

        if mode == 0:
            self._clear_line_from_cursor()
            self.lines = self.lines[: self.row + 1]
            return

        if mode == 1:
            for idx in range(0, self.row):
                self.lines[idx] = []
            self._clear_line_to_cursor()

    def _erase_chars(self, count: int) -> None:
        size = max(1, count)
        self._ensure_col(min(self.cols, self.col + size))
        line = self.lines[self.row]
        for idx in range(0, size):
            pos = self.col + idx
            if pos < len(line):
                line[pos] = " "

    def _insert_chars(self, count: int) -> None:
        size = max(1, count)
        self._ensure_col(self.col)
        line = self.lines[self.row]
        line[self.col : self.col] = [" "] * size
        if len(line) > self.cols:
            del line[self.cols :]

    def _delete_chars(self, count: int) -> None:
        size = max(1, count)
        line = self.lines[self.row]
        if self.col < len(line):
            del line[self.col : self.col + size]

    def _insert_lines(self, count: int) -> None:
        size = max(1, count)
        self.lines[self.row : self.row] = [[] for _ in range(size)]
        self._trim_scrollback()

    def _delete_lines(self, count: int) -> None:
        size = max(1, count)
        del self.lines[self.row : self.row + size]
        self._ensure_row(self.row)

    def _apply_csi(self, command: str, params_raw: str, responses: bytearray) -> None:
        params = params_raw.split(";") if params_raw else []

        if command == "K":
            mode = _parse_numeric_param(params[0] if params else None, 0)
            if mode == 0:
                self._clear_line_from_cursor()
            elif mode == 1:
                self._clear_line_to_cursor()
            elif mode == 2:
                self._clear_current_line()
            return

        if command == "J":
            self._clear_display(_parse_numeric_param(params[0] if params else None, 0))
            return

        if command == "X":
            self._erase_chars(_parse_numeric_param(params[0] if params else None, 1))
            return

        if command == "@":
            self._insert_chars(_parse_numeric_param(params[0] if params else None, 1))
            return

        if command == "P":
            self._delete_chars(_parse_numeric_param(params[0] if params else None, 1))
            return

        if command == "L":
            self._insert_lines(_parse_numeric_param(params[0] if params else None, 1))
            return

        if command == "M":
            self._delete_lines(_parse_numeric_param(params[0] if params else None, 1))
            return

        if command == "G":
            col = max(1, _parse_numeric_param(params[0] if params else None, 1))
            self.col = min(self.cols, col - 1)
            return

        if command == "C":
            move = max(1, _parse_numeric_param(params[0] if params else None, 1))
            self.col = min(self.cols, self.col + move)
            return

        if command == "D":
            move = max(1, _parse_numeric_param(params[0] if params else None, 1))
            self.col = max(0, self.col - move)
            return

        if command == "A":
            move = max(1, _parse_numeric_param(params[0] if params else None, 1))
            self.row = max(0, self.row - move)
            self._ensure_row(self.row)
            return

        if command == "B":
            move = max(1, _parse_numeric_param(params[0] if params else None, 1))
            self.row += move
            self._ensure_row(self.row)
            self._trim_scrollback()
            return

        if command == "E":
            move = max(1, _parse_numeric_param(params[0] if params else None, 1))
            self.row += move
            self.col = 0
            self._ensure_row(self.row)
            self._trim_scrollback()
            return

        if command == "F":
            move = max(1, _parse_numeric_param(params[0] if params else None, 1))
            self.row = max(0, self.row - move)
            self.col = 0
            self._ensure_row(self.row)
            return

        if command in {"H", "f"}:
            row = max(1, _parse_numeric_param(params[0] if params else None, 1))
            col = max(1, _parse_numeric_param(params[1] if len(params) > 1 else None, 1))
            self.row = row - 1
            self.col = min(self.cols, col - 1)
            self._ensure_row(self.row)
            self._trim_scrollback()
            return

        if command == "s":
            self.saved_row = self.row
            self.saved_col = self.col
            return

        if command == "u":
            self.row = max(0, self.saved_row)
            self.col = max(0, min(self.cols, self.saved_col))
            self._ensure_row(self.row)
            return

        if command == "n":
            mode = _parse_numeric_param(params[0] if params else None, 0)
            if mode == 6:
                responses.extend(f"\x1b[{self.row + 1};{self.col + 1}R".encode("ascii"))
            elif mode == 5:
                responses.extend(b"\x1b[0n")
            return

        if command == "c":
            if params_raw.startswith(">"):
                responses.extend(b"\x1b[>0;0;0c")
            else:
                responses.extend(b"\x1b[?1;2c")
            return

        if command in {"m", "h", "l"}:
            return


class TerminalSession:
    """Incremental PTY stream decoder + terminal screen state."""

    def __init__(self, *, cols: int = DEFAULT_COLS, rows: int = DEFAULT_ROWS) -> None:
        self._emulator = TerminalEmulator(cols=cols)
        self._rows = _clamp_rows(rows)
        self._decoder = codecs.getincrementaldecoder("utf-8")("replace")
        self._seq = 0

    @property
    def cols(self) -> int:
        return self._emulator.cols

    @property
    def rows(self) -> int:
        return self._rows

    def resize(self, *, cols: int, rows: int) -> TerminalFrame:
        self._emulator.set_cols(cols)
        self._rows = _clamp_rows(rows)
        return self._frame()

    def snapshot(self) -> TerminalFrame:
        return self._frame()

    def feed_bytes(self, data: bytes) -> TerminalUpdate:
        text = self._decoder.decode(data)
        responses = self._emulator.feed(text)
        return TerminalUpdate(frame=self._frame(), responses=responses)

    def feed_text(self, text: str) -> TerminalUpdate:
        responses = self._emulator.feed(text)
        return TerminalUpdate(frame=self._frame(), responses=responses)

    def _frame(self) -> TerminalFrame:
        self._seq += 1
        return TerminalFrame(
            seq=self._seq,
            text=self._emulator.text(),
            cols=self._emulator.cols,
            rows=self._rows,
            cursor_row=self._emulator.row,
            cursor_col=self._emulator.col,
        )
