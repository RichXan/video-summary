import json
import tempfile
import time
import unittest
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
import prepare_media_jiji


class PrepareMediaJijiTest(unittest.TestCase):
    def test_load_netscape_cookies_keeps_latest_value_by_name(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "cookies.txt"
            path.write_text(
                "\n".join(
                    [
                        "# Netscape HTTP Cookie File",
                        ".douyin.com\tTRUE\t/\tTRUE\t1810000000\tttwid\told",
                        "www.douyin.com\tFALSE\t/\tFALSE\t1810000000\ts_v_web_id\tweb",
                        ".douyin.com\tTRUE\t/\tTRUE\t1810000000\tttwid\tnew",
                    ]
                ),
                encoding="utf-8",
            )

            cookies = prepare_media_jiji.load_cookie_file(path)

            self.assertEqual(cookies["ttwid"], "new")
            self.assertEqual(cookies["s_v_web_id"], "web")

    def test_load_json_cookie_export(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "cookies.json"
            path.write_text(
                json.dumps(
                    {
                        "cookies": [
                            {"name": "sessionid", "value": "abc", "domain": ".douyin.com"},
                            {"name": "ttwid", "value": "def", "domain": ".douyin.com"},
                        ]
                    }
                ),
                encoding="utf-8",
            )

            cookies = prepare_media_jiji.load_cookie_file(path)

            self.assertEqual(cookies["sessionid"], "abc")
            self.assertEqual(cookies["ttwid"], "def")

    def test_find_newest_mp4_returns_most_recent_file(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            old = root / "old.mp4"
            new = root / "nested" / "new.mp4"
            new.parent.mkdir()
            old.write_bytes(b"old")
            new.write_bytes(b"new")
            now = time.time()
            old_time = now - 100
            new_time = now
            old.touch()
            new.touch()
            old_stat = (old_time, old_time)
            new_stat = (new_time, new_time)
            import os

            os.utime(old, old_stat)
            os.utime(new, new_stat)

            self.assertEqual(prepare_media_jiji.find_newest_mp4(root), new)

    def test_extracts_url_from_douyin_share_text(self):
        text = "复制此链接，打开抖音 https://v.douyin.com/A1_FtldGgIw/ 直接观看视频！"

        url = prepare_media_jiji.extract_first_url(text)

        self.assertEqual(url, "https://v.douyin.com/A1_FtldGgIw/")

    def test_canonicalizes_share_video_url(self):
        url = "https://www.iesdouyin.com/share/video/7655554393391537802/?region=CN"

        canonical = prepare_media_jiji.canonical_douyin_url(url)

        self.assertEqual(canonical, "https://www.douyin.com/video/7655554393391537802")

    def test_extract_metadata_from_downloaded_json(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            data = root / "video_data.json"
            data.write_text(
                json.dumps(
                    {
                        "item_title": "How to avoid dollar trouble?",
                        "author": {"nickname": "Creator"},
                        "duration": 503000,
                    }
                ),
                encoding="utf-8",
            )

            metadata = prepare_media_jiji.extract_metadata(root)

            self.assertEqual(metadata["title"], "How to avoid dollar trouble?")
            self.assertEqual(metadata["author"], "Creator")
            self.assertEqual(metadata["duration"], 503)


if __name__ == "__main__":
    unittest.main()
