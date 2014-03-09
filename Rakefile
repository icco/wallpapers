require 'padrino-core/cli/rake'

PadrinoTasks.use(:database)
PadrinoTasks.init

PROD = true

# Local Path to sync
PATH = "#{ENV['HOME']}/Dropbox/Photos/Wallpapers/DesktopWallpapers"

desc "Run a local server."
task :local do
  Kernel.exec("shotgun -s thin -p 9393")
end

desc "Sync local files with GCS."
task :push do
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

  total = created + updated - deleted
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
