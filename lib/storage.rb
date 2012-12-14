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

  def self.directory directory_name, force_prod = false
    directory = self.connection(force_prod).directories.get(directory_name)

    if directory.nil?
      directory = self.connection.directories.create(
        :key    => directory_name,
        :public => true
      )
    end

    return directory
  end

  def self.main_dir force_prod = false
    return self.directory "iccowalls", force_prod
  end

  def self.thumb_dir force_prod = false
    return self.directory "iccothumbs", force_prod
  end

  def self.get_files force_prod = false
    return self.main_dir(force_prod).files
  end

  def self.get_range range, force_prod = false
    return self.get_files(force_prod).to_a()[range]
  end

  def self.get_file filename, force_prod = false
    return self.get_files(force_prod).get(filename)
  end

  def self.get_thumb filename, force_prod = false
    return self.thumb_dir(force_prod).files.get(filename)
  end
end
