class Storage
  def self.connection force_prod = false
    directory_name = "iccowalls"

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

    connection = Fog::Storage.new(credentials)

    directory = connection.directories.get(directory_name)

    if directory.nil?
      directory = connection.directories.create(
        :key    => directory_name,
        :public => true
      )
    end

    return directory
  end

  def self.get_files
    return self.connection.files
  end

  def self.get_file filename
    return self.get_files.get(filename)
  end
end
