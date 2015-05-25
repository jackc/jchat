require 'ffaker'
require 'scrypt'

module FactoryHelper
  def create_user attrs={}
    defaults = {
      email: FFaker::Internet.email,
      name: FFaker::Name.first_name.gsub(/\W/, ''),
      password: "password"
    }
    attrs = defaults.merge(attrs)
    attrs[:password_salt] = Sequel::SQL::Blob.new "salt"
    password_digest = SCrypt::Engine.__sc_crypt attrs.delete(:password), attrs[:password_salt], 16384, 8, 1, 32
    attrs[:password_digest] = Sequel::SQL::Blob.new password_digest
    DB[:users].insert attrs
  end

  def create_channel attrs={}
    defaults = { name: FFaker::Movie.title }
    attrs = defaults.merge(attrs)
    DB[:channels].insert attrs
  end
end
