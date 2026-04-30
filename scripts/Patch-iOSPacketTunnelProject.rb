#!/usr/bin/env ruby
# frozen_string_literal: true

require 'optparse'
require 'pathname'
require 'xcodeproj'

options = {
  packet_target: 'PacketTunnel',
  deployment_target: '15.0',
  app_bundle_id: 'github.com.observerss.detour2fyne',
  packet_bundle_id: 'github.com.observerss.detour2fyne.PacketTunnel',
  app_group: 'group.github.com.observerss.detour2fyne'
}

OptionParser.new do |parser|
  parser.banner = 'usage: scripts/Patch-iOSPacketTunnelProject.rb --project App.xcodeproj --stage-dir Detour2Native [options]'
  parser.on('--project PATH', 'Xcode project to patch') { |value| options[:project] = value }
  parser.on('--stage-dir PATH', 'Prepared native overlay directory') { |value| options[:stage_dir] = value }
  parser.on('--app-target NAME', 'Containing app target name') { |value| options[:app_target] = value unless value.to_s.empty? }
  parser.on('--packet-target NAME', 'Packet Tunnel target name') { |value| options[:packet_target] = value unless value.to_s.empty? }
  parser.on('--app-bundle-id ID', 'Containing app bundle id') { |value| options[:app_bundle_id] = value }
  parser.on('--packet-bundle-id ID', 'Packet Tunnel bundle id') { |value| options[:packet_bundle_id] = value }
  parser.on('--app-group ID', 'App Group id') { |value| options[:app_group] = value }
  parser.on('--deployment-target VERSION', 'iOS deployment target') { |value| options[:deployment_target] = value }
end.parse!

abort '--project is required' if options[:project].to_s.empty?
abort '--stage-dir is required' if options[:stage_dir].to_s.empty?

project_path = Pathname.new(options[:project]).expand_path
stage_dir = Pathname.new(options[:stage_dir]).expand_path
abort "Xcode project not found: #{project_path}" unless project_path.directory?
abort "stage directory not found: #{stage_dir}" unless stage_dir.directory?

project_dir = project_path.dirname
stage_relative_path = stage_dir.relative_path_from(project_dir).to_s
project = Xcodeproj::Project.open(project_path.to_s)

def native_targets(project)
  project.targets.select { |target| target.respond_to?(:product_type) }
end

def find_app_target(project, target_name)
  if target_name && !target_name.empty?
    target = project.targets.find { |candidate| candidate.name == target_name }
    abort "app target not found: #{target_name}" unless target
    return target
  end

  target = native_targets(project).find { |candidate| candidate.product_type == 'com.apple.product-type.application' }
  target || native_targets(project).first || abort('no native app target found')
end

def find_or_create_group(project, stage_relative_path)
  group = project.main_group.groups.find { |candidate| candidate.path == stage_relative_path || candidate.name == 'Detour2Native' }
  return group if group

  group = project.main_group.new_group('Detour2Native', stage_relative_path)
  group.source_tree = '<group>'
  group
end

def ensure_file(group, relative_path)
  existing = group.files.find { |file_ref| file_ref.path == relative_path }
  existing || group.new_file(relative_path)
end

def ensure_system_framework(project, framework_name)
  frameworks_group = project.frameworks_group || project.main_group.new_group('Frameworks')
  framework_path = "System/Library/Frameworks/#{framework_name}.framework"
  existing = frameworks_group.files.find { |file_ref| file_ref.path == framework_path }
  return existing if existing

  file_ref = frameworks_group.new_file(framework_path)
  file_ref.source_tree = 'SDKROOT'
  file_ref
end

def add_to_sources(target, file_ref)
  return if target.source_build_phase.files.any? { |build_file| build_file.file_ref == file_ref }

  target.source_build_phase.add_file_reference(file_ref, true)
end

def add_to_frameworks(target, file_ref)
  return if target.frameworks_build_phase.files.any? { |build_file| build_file.file_ref == file_ref }

  target.frameworks_build_phase.add_file_reference(file_ref, true)
end

def append_build_setting(settings, key, value)
  current = settings[key]
  values = case current
           when nil
             []
           when Array
             current
           else
             [current]
           end
  settings[key] = (values + [value]).uniq
end

def configure_build_settings(target, values)
  target.build_configurations.each do |configuration|
    values.each do |key, value|
      configuration.build_settings[key] = value
    end
  end
end

def append_target_build_setting(target, key, value)
  target.build_configurations.each do |configuration|
    append_build_setting(configuration.build_settings, key, value)
  end
end

def ensure_copy_phase(target, phase_name, destination_spec)
  phase = target.copy_files_build_phases.find { |candidate| candidate.name == phase_name }
  return phase if phase

  phase = target.project.new(Xcodeproj::Project::Object::PBXCopyFilesBuildPhase)
  phase.name = phase_name
  phase.dst_subfolder_spec = destination_spec
  target.build_phases << phase
  phase
end

def add_embed_product_phase(app_target, product_ref)
  phase = ensure_copy_phase(app_target, 'Embed App Extensions', '13')
  existing = phase.files.find { |build_file| build_file.file_ref == product_ref }
  return if existing

  build_file = phase.add_file_reference(product_ref, true)
  build_file.settings = { 'ATTRIBUTES' => ['RemoveHeadersOnCopy'] }
end

def add_target_dependency(app_target, dependency_target)
  return if app_target.dependencies.any? { |dependency| dependency.target == dependency_target }

  app_target.add_dependency(dependency_target)
end

app_target = find_app_target(project, options[:app_target])
packet_target = project.targets.find { |target| target.name == options[:packet_target] }
unless packet_target
  packet_target = project.new_target(:app_extension, options[:packet_target], :ios, options[:deployment_target])
  packet_target.product_reference.path = "#{options[:packet_target]}.appex"
  packet_target.product_reference.explicit_file_type = 'wrapper.app-extension'
end

native_group = find_or_create_group(project, stage_relative_path)
packet_sources = [
  'PacketTunnel/PacketTunnelProvider.swift',
  'PacketTunnel/Tun2SocksRunner.swift',
  'PacketTunnel/GomobileBackend.swift',
  'PacketTunnel/PacketFlowAdapter.swift',
  'Native/DetourTunnelConfig.swift'
]
app_sources = [
  'Native/DetourTunnelManager.swift',
  'Native/DetourTunnelConfig.swift'
]

packet_sources.each { |source_path| add_to_sources(packet_target, ensure_file(native_group, source_path)) }
app_sources.each { |source_path| add_to_sources(app_target, ensure_file(native_group, source_path)) }

network_extension_ref = ensure_system_framework(project, 'NetworkExtension')
add_to_frameworks(app_target, network_extension_ref)
add_to_frameworks(packet_target, network_extension_ref)

gomobile_ref = ensure_file(native_group, 'Frameworks/DetourTunnelBackend.xcframework')
add_to_frameworks(packet_target, gomobile_ref)

configure_build_settings(app_target, {
  'PRODUCT_BUNDLE_IDENTIFIER' => options[:app_bundle_id],
  'CODE_SIGN_ENTITLEMENTS' => "#{stage_relative_path}/Detour2.entitlements",
  'SWIFT_VERSION' => '5.0',
  'IPHONEOS_DEPLOYMENT_TARGET' => options[:deployment_target]
})
append_target_build_setting(app_target, 'FRAMEWORK_SEARCH_PATHS', '$(inherited)')
append_target_build_setting(app_target, 'FRAMEWORK_SEARCH_PATHS', "$(PROJECT_DIR)/#{stage_relative_path}/Frameworks")

configure_build_settings(packet_target, {
  'PRODUCT_BUNDLE_IDENTIFIER' => options[:packet_bundle_id],
  'INFOPLIST_FILE' => "#{stage_relative_path}/PacketTunnel/Info.plist",
  'CODE_SIGN_ENTITLEMENTS' => "#{stage_relative_path}/PacketTunnel/PacketTunnel.entitlements",
  'SWIFT_VERSION' => '5.0',
  'IPHONEOS_DEPLOYMENT_TARGET' => options[:deployment_target],
  'APPLICATION_EXTENSION_API_ONLY' => 'YES',
  'PRODUCT_NAME' => '$(TARGET_NAME)',
  'LD_RUNPATH_SEARCH_PATHS' => ['$(inherited)', '@executable_path/Frameworks', '@executable_path/../../Frameworks']
})
append_target_build_setting(packet_target, 'FRAMEWORK_SEARCH_PATHS', '$(inherited)')
append_target_build_setting(packet_target, 'FRAMEWORK_SEARCH_PATHS', "$(PROJECT_DIR)/#{stage_relative_path}/Frameworks")

add_embed_product_phase(app_target, packet_target.product_reference)
add_target_dependency(app_target, packet_target)

project.save

puts "[ok] patched #{project_path}"
puts "[ok] app target: #{app_target.name}"
puts "[ok] packet tunnel target: #{packet_target.name}"
puts "[ok] native stage: #{stage_dir}"