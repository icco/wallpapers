require File.expand_path('../config/boot.rb', __FILE__)
require 'padrino-core/cli/rake'

PROD = true

# Local Path to sync
PATH = "#{ENV['HOME']}/Dropbox/Photos/Wallpapers/DesktopWallpapers"

PadrinoTasks.init

desc "Run a local server."
task :local do
  Kernel.exec("shotgun -s thin -p 9393")
end

desc "Sync local files with GCS."
task :push do

  dir = Storage.main_dir PROD

  local = Dir.open(PATH).to_a.delete_if {|f| f.start_with? '.' }
  remote = Storage.get_files PROD

  remote.each do |file|
    filename = file.key.gsub("\+", " ")
    if !local.include? filename
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

      puts "created"
    else
      file.body = File.open("#{PATH}/#{filename}")
      file.save

      puts "updated"
    end
  end
end

desc "Erase all thumbnails."
task :purge_thumbnails do
  dir = Storage.thumb_dir PROD
  dir.files.each do |file|
    file.destroy
    puts "#{file.key} - deleted"
  end
end
