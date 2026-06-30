import sys
import unittest
from pathlib import Path
from types import SimpleNamespace

sys.path.insert(0, str(Path(__file__).resolve().parent))
import transcribe_faster_whisper


class TranscribeFasterWhisperTest(unittest.TestCase):
    def test_segments_to_payload_trims_empty_text(self):
        segments = [
            SimpleNamespace(start=0.0, end=1.5, text=" hello "),
            SimpleNamespace(start=1.5, end=2.0, text="  "),
            SimpleNamespace(start=2.0, end=3.0, text="world"),
        ]

        payload = transcribe_faster_whisper.segments_to_payload(segments)

        self.assertEqual(
            payload,
            [
                {"start": 0.0, "end": 1.5, "text": "hello"},
                {"start": 2.0, "end": 3.0, "text": "world"},
            ],
        )


if __name__ == "__main__":
    unittest.main()
