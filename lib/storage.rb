class Storage
  def self.connection
    directory_name = "iccowalls"

    if Padrino.env == :development
      credentials = {
        :provider   => "Local",
        :local_root => "/tmp/",
        :endpoint   => "file:///tmp/",
      }
    else
      credentials = {
        :provider                         => 'Google',
        :google_storage_access_key_id     => ENV['GOOGLE_KEY'],
        :google_storage_secret_access_key => ENV['GOOGLE_SECRET'],
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
    return storage_connection.files
  end

  def self.get_file filename
    return get_files.get(filename)
  end
end
