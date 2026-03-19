#!/usr/bin/env python3
import os
import sys


def parse_bool(raw: str, default: bool) -> bool:
    v = (raw or "").strip().lower()
    if v in {"1", "true", "yes", "on"}:
        return True
    if v in {"0", "false", "no", "off"}:
        return False
    return default


def parse_int(raw: str, default: int) -> int:
    try:
        return int((raw or "").strip())
    except Exception:
        return default


def main() -> int:
    audio_path = (os.getenv("KNIT_STT_AUDIO_PATH") or "").strip()
    if not audio_path:
        print("KNIT_STT_AUDIO_PATH is required", file=sys.stderr)
        return 2
    if not os.path.isfile(audio_path):
        print(f"audio file not found: {audio_path}", file=sys.stderr)
        return 2

    model_name = (os.getenv("KNIT_FASTER_WHISPER_MODEL") or "small").strip() or "small"
    device = (os.getenv("KNIT_FASTER_WHISPER_DEVICE") or "cpu").strip() or "cpu"
    compute_type = (os.getenv("KNIT_FASTER_WHISPER_COMPUTE_TYPE") or "int8").strip() or "int8"
    language = (os.getenv("KNIT_FASTER_WHISPER_LANGUAGE") or "").strip() or None
    beam_size = max(1, parse_int(os.getenv("KNIT_FASTER_WHISPER_BEAM_SIZE"), 1))
    vad_filter = parse_bool(os.getenv("KNIT_FASTER_WHISPER_VAD_FILTER"), True)
    condition_on_prev = parse_bool(os.getenv("KNIT_FASTER_WHISPER_CONDITION_ON_PREV_TEXT"), False)

    try:
        from faster_whisper import WhisperModel
    except Exception as exc:
        print(f"failed to import faster_whisper: {exc}", file=sys.stderr)
        return 2

    try:
        model = WhisperModel(model_name, device=device, compute_type=compute_type)
        segments, _ = model.transcribe(
            audio_path,
            beam_size=beam_size,
            language=language,
            vad_filter=vad_filter,
            condition_on_previous_text=condition_on_prev,
        )
        parts = []
        for seg in segments:
            text = (seg.text or "").strip()
            if text:
                parts.append(text)
        transcript = " ".join(parts).strip()
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
