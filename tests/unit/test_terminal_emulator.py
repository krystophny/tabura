from __future__ import annotations

import pytest

from tabula.web.terminal_emulator import TerminalEmulator, TerminalSession


@pytest.mark.parametrize(
    ("raw", "expected"),
    [
        ("line1\r\nline2\r\n", "line1\nline2\n"),
        ("abcdef", "abcdef"),
        ("10%\r20%\r100%\n", "100%\n"),
        ("booting\r\x1b[2Kready\r\nnext\r\x1b[2Kdone\r\n", "ready\ndone\n"),
        ("abc\rX\n", "Xbc\n"),
        ("\x1b]0;title\x07\x1b[31mred\x1b[0m\r\nok\n", "red\nok\n"),
        ("processing file 123\r\x1b[2Kdone\n", "done\n"),
        ("hello world\r\x1b[6C\x1b[5X\n", "hello      \n"),
        ("abcd\r\x1b[2C\x1b[@X\n", "abXcd\n"),
        ("abcdef\r\x1b[2C\x1b[2P\n", "abef\n"),
        ("ab\bX\n", "aX\n"),
        ("\x1b(0lqk\x1b(B\n", "lqk\n"),
        ("line1\nline2\n\x1b[1A\r\x1b[2KlineX\n", "line1\nlineX\n"),
        ("a\nb\nc\x1b[1A\r\x1b[0JX\n", "a\nX\n"),
        ("one\ntwo\n\x1b[2Jthree\n", "one\ntwo\n\nthree\n"),
        ("aaaa\nbbbb\ncccc\x1b[2;2HZZ\n", "aaaa\nbZZb\ncccc"),
        ("line1\nline2\nline3\x1b[2;1H\x1b[Lnew\n", "line1\nnew\nline2\nline3"),
        ("line1\nline2\nline3\x1b[2;1H\x1b[M", "line1\nline3"),
        ("abc\x1b[s123\x1b[uZ\n", "abcZ23\n"),
        ("a\tb\n", "a       b\n"),
        ("abcde\r\x1b[3C\x1b[1K\n", "   de\n"),
        ("abc\ndef\x1b[2;2H\x1b[0JX\n", "abc\ndX\n"),
        ("abc\ndef\x1b[2;2H\x1b[1JX\n", "\n Xf\n"),
        ("one\ntwo\n\x1b[3Jthree\n", "three\n"),
        ("abc\x1b[1GZ\n", "Zbc\n"),
        ("abc\r\x1b[5C\x1b[2DZ\n", "abcZ\n"),
        ("top\x1b[2B\x1b[1Ebottom\n", "top\n\n\nbottom\n"),
        ("line1\nline2\n\x1b[1FZ\n", "line1\nZine2\n"),
        ("abc\x1b[?;fZ\n", "Zbc\n"),
        ("abc\x1b7XX\x1b8Z\x1bDa\x1bEb\n", "abcZX\n    a\nb\n"),
        ("ab\ncd\x1b[2;1H\x1bMX\x1bcnew\n", "new\n"),
        ("ab\x1b#cd\x1b(", "abcd"),
        ("abc\x1b[31", "abc"),
        ("a\x01\x7fb\n", "ab\n"),
        ("a\r\x1b[-2CZ\n", "aZ\n"),
    ],
)
def test_terminal_emulator_render_examples(raw: str, expected: str) -> None:
    em = TerminalEmulator(cols=120, tab_size=8)
    em.feed(raw)
    assert em.text() == expected


def test_terminal_emulator_col_wrap() -> None:
    em = TerminalEmulator(cols=4)
    em.feed("abcdef")
    assert em.text() == "abcd\nef"


def test_terminal_emulator_tab_size() -> None:
    em = TerminalEmulator(tab_size=4)
    em.feed("a\tb\n")
    assert em.text() == "a   b\n"


def test_terminal_emulator_csi_2j_without_scrollback_preservation() -> None:
    em = TerminalEmulator(preserve_scrollback_on_clear=False)
    em.feed("one\ntwo\n\x1b[2Jthree\n")
    assert em.text() == "three\n"


def test_terminal_emulator_insert_characters_trims_at_column_limit() -> None:
    em = TerminalEmulator(cols=6)
    em.feed("abcd\r\x1b[2C\x1b[4@Z")
    assert em.text() == "abZ   "


def test_terminal_session_reports_terminal_queries() -> None:
    session = TerminalSession(cols=80, rows=24)
    update = session.feed_text("abc\x1b[6n\x1b[c\x1b[>c\x1b]10;?\x1b\\\x1b]11;?\x1b\\")
    assert update.responses.startswith(b"\x1b[1;4R\x1b[?1;2c\x1b[>0;0;0c")
    assert b"\x1b]10;rgb:cccc/cccc/cccc\x1b\\" in update.responses
    assert b"\x1b]11;rgb:0000/0000/0000\x1b\\" in update.responses


def test_terminal_session_payload_shape() -> None:
    session = TerminalSession(cols=120, rows=40)
    frame = session.snapshot().to_payload()
    assert frame["type"] == "terminal_frame"
    assert frame["screen"]["text"] == ""
    assert frame["screen"]["format_spans"] == []
