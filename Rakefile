require 'padrino-core/cli/rake'

require File.expand_path('../config/boot.rb', __FILE__)

PadrinoTasks.use(:database)
PadrinoTasks.use(:activerecord)
PadrinoTasks.init

PROD = true

# Local Path to sync
PATH = "#{ENV['HOME']}/Dropbox/Photos/Wallpapers/DesktopWallpapers"

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
task :push => [:environment, :clean] do
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
