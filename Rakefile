require File.expand_path('../config/boot.rb', __FILE__)
require 'padrino-core/cli/rake'

PadrinoTasks.init

desc "Run a local server."
task :local do
  Kernel.exec("shotgun -s thin -p 9393")
end

desc "Sync local files with GCS."
task :push do
  dir = Storage.connection true
  path = "#{ENV['HOME']}/Dropbox/Photos/Wallpapers/DesktopWallpapers"
  local = Dir.open(path).to_a.delete_if {|f| f.start_with? '.' }

  local.each do |filename|
    print "#{filename} - "

    file = dir.files.get filename

    if file.nil?
      file = dir.files.create(
        :key    => filename,
        :body   => File.open("#{path}/#{filename}"),
        :public => true,
      )

      puts "created"
    else
      file.body = File.open("#{path}/#{filename}")
      file.save

      puts "updated"
    end
  end
end
