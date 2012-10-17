class Storage
  def self.connection force_prod = false
    if Padrino.env != :development or force_prod
      credentials = {
        :provider                         => 'Google',
        :google_storage_access_key_id     => ENV['GOOGLE_KEY'],
        :google_storage_secret_access_key => ENV['GOOGLE_SECRET'],
      }
    else
      credentials = {
        :provider   => "Local",
        :local_root => "/tmp/",
        :endpoint   => "file:///tmp/",
      }
    end

    return Fog::Storage.new(credentials)
  end

  def self.directory directory_name
    directory = self.connection.directories.get(directory_name)

    if directory.nil?
      directory = self.connection.directories.create(
        :key    => directory_name,
        :public => true
      )
    end

    return directory
  end

  def self.main_dir
    return self.directory "iccowalls"
  end

  def self.thumb_dir
    return self.directory "iccothumbs"
  end

  def self.get_files
    return self.main_dir.files
  end

  def self.get_file filename
    return self.get_files.get(filename)
  end

  def self.get_thumb filename
    return self.thumb_dir.files.get(filename)
  end
end
