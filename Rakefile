require "rubygems"
require "bundler"
Bundler.require(:default, ENV["RACK_ENV"] || :development)
require "./lib/storage.rb"

# Local Path to sync
PATH = "#{ENV['HOME']}/Dropbox/Photos/Wallpapers/DesktopWallpapers"
PROD = true

desc "Run a local server."
task :local do
  Kernel.exec("shotgun -s thin -p 9393")
end

desc "Clean filenames of all images."
task :clean do
  local = Dir.open(PATH).to_a.delete_if {|f| f.start_with? '.' }
  local.each do |old_filename|
    ext = File.extname(old_filename)
    name = File.basename(old_filename, ext)

    new_filename = "#{PATH}/#{name.downcase.gsub(/[^a-z0-9]/, '')}#{ext.downcase}"
    old_filename = "#{PATH}/#{old_filename}"

    change = !old_filename.eql?(new_filename)

    if change
      puts "#{old_filename} => #{new_filename}"
      File.rename(old_filename, new_filename)
    end
  end
end

desc "Sync local files with GCS."
task :push => [:clean] do
  deleted = 0
  created = 0
  updated = 0

  dir = Storage.main_dir PROD

  local = Dir.open(PATH).to_a.delete_if {|f| f.start_with? '.' }
  remote = Storage.get_files PROD

  remote.each do |file|
    filename = file.key.gsub("\+", " ")
    if !local.include? filename
      deleted += 1
      puts "#{filename} - deleted"
      file.destroy
    end
  end

  local.each do |filename|
    print "#{filename} - "

    file = dir.files.get filename

    if file.nil?
      file = dir.files.create(
        :key    => filename,
        :body   => File.open("#{PATH}/#{filename}"),
        :public => true,
      )

      created += 1
      puts "created"
    else
      file.body = File.open("#{PATH}/#{filename}")
      file.public = true
      file.save

      updated += 1
      puts "updated"
    end
  end

  total = (created + updated) - deleted
  puts """
Stats:

  Created: #{created}
  Updated: #{updated}
  Deleted: #{deleted}
  -------------------
  Total:   #{total}

  """
end

desc "Erase all thumbnails."
task :purge_thumbnails do
  dir = Storage.thumb_dir PROD
  dir.files.each do |file|
    file.destroy
    puts "#{file.key} - deleted"
  end
end

desc "Generate all thumbnails."
task :generate_thumbs do
  dir = Storage.thumb_dir PROD
  images = Storage.get_files PROD

  images.each do |basefile|
    thumbnail = MiniMagick::Image.read(basefile.body)
    thumbnail.combine_options do |c|
      c.quality "60"
      c.resize "600x400"
    end

    thumbnail_file = File.join(File.dirname(__FILE__), "tmp", "thumb", basefile.key)
    thumbnail.write thumbnail_file
    file = Storage.thumb_dir(PROD).files.create(
      :key    => basefile.key,
      :body   => File.open(thumbnail_file),
      :public => true,
    )

    puts "#{basefile.file_url} -> #{file.thumb_url}"
  end
end
