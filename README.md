# Wallpapers

A [site](http://wallpapers.natwelch.com) that lists the photos currently in my Wallpaper rotation. Hosts the images in Google Cloud Storage, and the site on Heroku. Eventually, I'd like it to pull from Dropbox instead.

## Design

 * `rake sync`

Pull look at wallpaper folder, and sync it with Storage service.

 * `site.rb`

A simple photogallery. Shows all of the photos, links to download. Should also create thumbnails automatically.
