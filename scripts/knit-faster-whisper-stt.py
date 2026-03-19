#!/usr/bin/env python3
"""
Local faster-whisper transcription entrypoint for Knit.

This script reads the audio path from KNIT_STT_AUDIO_PATH, runs faster-whisper,
and prints plain transcript text to stdout.
"""

from __future__ import annotations

import os
import sys
from typing import Optional


def parse_bool(raw: str, default: bool) -> bool:
    value = (raw or "").strip().lower()
    if value in {"1", "true", "yes", "on"}:
        return True
    if value in {"0", "false", "no", "off"}:
        return False
    return default


def env_int(name: str, default: int) -> int:
    raw = os.getenv(name, "").strip()
    if not raw:
        return default
    try:
        return int(raw)
    except ValueError:
        return default


def main() -> int:
    audio_path = os.getenv("KNIT_STT_AUDIO_PATH", "").strip()
    if not audio_path:
        print("KNIT_STT_AUDIO_PATH is required", file=sys.stderr)
        return 2
    if not os.path.isfile(audio_path):
        print(f"audio file not found: {audio_path}", file=sys.stderr)
        return 2

    model_name = os.getenv("KNIT_FASTER_WHISPER_MODEL", "small").strip() or "small"
    device = os.getenv("KNIT_FASTER_WHISPER_DEVICE", "cpu").strip() or "cpu"
    compute_type = os.getenv("KNIT_FASTER_WHISPER_COMPUTE_TYPE", "int8").strip() or "int8"
    beam_size = env_int("KNIT_FASTER_WHISPER_BEAM_SIZE", 1)
    language: Optional[str] = os.getenv("KNIT_FASTER_WHISPER_LANGUAGE", "").strip() or None
    vad_filter = parse_bool(os.getenv("KNIT_FASTER_WHISPER_VAD_FILTER", "true"), True)
    condition_on_previous_text = parse_bool(
        os.getenv("KNIT_FASTER_WHISPER_CONDITION_ON_PREV_TEXT", "false"),
        False,
    )

    try:
        from faster_whisper import WhisperModel  # type: ignore
    except Exception as exc:
        print(f"failed to import faster_whisper: {exc}", file=sys.stderr)
        return 2

    try:
        model = WhisperModel(model_name, device=device, compute_type=compute_type)
        segments, _info = model.transcribe(
            audio_path,
            beam_size=max(1, beam_size),
            language=language,
            vad_filter=vad_filter,
            condition_on_previous_text=condition_on_previous_text,
        )
        text_parts = []
        for segment in segments:
            chunk = (segment.text or "").strip()
            if chunk:
                text_parts.append(chunk)
        transcript = " ".join(text_parts).strip()
        if not transcript:
            print("empty transcript", file=sys.stderr)
            return 1
        print(transcript)
        return 0
    except Exception as exc:
        print(f"faster-whisper transcription failed: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
