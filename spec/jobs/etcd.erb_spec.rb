# frozen_string_literal: true

require 'rspec'
require 'spec_helper'
require 'fileutils'
require 'open3'

describe 'etcd.erb' do
  let(:link_spec) do {
    'kube-apiserver' => {
      'instances' => [],
      'properties' => {
        'tls-cipher-suites' => 'TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384'
      }
    },
    'etcd' => {
      'properties' => { },
      'instances' => [ ]
    }
  }
  end

  let(:rendered_template) do
    compiled_template('etcd', 'bin/etcd', {}, link_spec, {}, 'z1', 'fake-bosh-ip', 'fake-bosh-id')
  end

  it 'includes default cipher-suites' do
    expect(rendered_template).to include('--cipher-suites=TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384')
  end
end
