# GoToSocial Thumbnailer

> Disclaimer: This software is written with my specific setup in mind. It might
> work with your setup but nothing is guaranteed. Most notably, this only works
> with S3 storage

Video support in
[GoToSocial](https://github.com/superseriousbusiness/gotosocial) works without
any external dependencies which is awesome! However this comes at the cost of
not having thumbnails. Instead of polluting upstream, the gotosocial-thumbnailer
retroactively processes video files and generates a fitting thumbnail using ffmpeg.
