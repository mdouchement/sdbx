package main

import (
	"fmt"
	"os/user"
)

// Some scripts are borrowed and adapted from https://github.com/89luca89/distrobox/blob/main/distrobox-init
// Under GPL-3.0 license

func SetupEnv(u *user.User) string {
	script := fmt.Sprintf(`
		#set -o xtrace
		container_user_home="%s"
		container_user_uid=%s
		container_user_gid=%s
		container_user_name="%s"
	`, u.HomeDir, u.Uid, u.Gid, u.Username)

	return TrimDoc(script)
}

func SetupSudoers() string {
	return TrimDoc(`
		printf "sdbx: Setting up sudo...\n"
		mkdir -p /etc/sudoers.d
		# Ensure we're using the user's password for sudo, not root
		if [ -e /etc/sudoers ]; then
					sed -i "s|^Defaults targetpw.*||g" /etc/sudoers
		fi

		# Do not check fqdn when doing sudo, it will not work anyways
		# Also allow canonical groups to use sudo
		cat << EOF > /etc/sudoers.d/sudoers
		Defaults !targetpw
		Defaults !fqdn
		%wheel ALL=(ALL:ALL) ALL
		%sudo ALL=(ALL:ALL) ALL
		%root ALL=(ALL:ALL) ALL
		EOF
	`)
}

func SetupUser() string {
	return TrimDoc(`
		if [ "${container_user_uid}" -ne 0 ]; then
			# Ensure passwordless sudo is set up for user
			printf "\"%%s\" ALL = (root) NOPASSWD:ALL\n" "${container_user_name}" >> /etc/sudoers.d/sudoers
		fi

		printf "sdbx: Setting up user's group list...\n"
		# If we have sudo/wheel groups, let's add the user to them.
		# and ensure that user's in those groups can effectively sudo
		additional_groups=""
		if grep -q "^sudo" /etc/group; then
			additional_groups="sudo"
		elif grep -q "^wheel" /etc/group; then
			additional_groups="wheel"
		elif grep -q "^root" /etc/group; then
			additional_groups="root"
		fi

		if ! grep -q "^${container_user_name}:" /etc/group; then
			printf "sdbx: Setting up user groups...\n"

			if ! groupadd --force --gid "${container_user_gid}" "${container_user_name}"; then
				# It may occur that we have users with unsupported user name (eg. on LDAP or AD)
				# So let's try and force the group creation this way.
				printf "%%s:x:%%s:\n" "${container_user_name}" "${container_user_gid}" >> /etc/group
			fi
		fi

		if ! grep -q "^$(printf '%s' "${container_user_name}" | tr '\\' '.'):" /etc/passwd &&
			! getent passwd "${container_user_uid}"; then
			printf "sdbx: Adding user...\n"
			if ! useradd \
				--home-dir "${container_user_home}" \
				--create-home \
				--groups "${additional_groups}" \
				--shell "${SHELL:-"/bin/bash"}" \
				--uid "${container_user_uid}" \
				--gid "${container_user_gid}" \
				"${container_user_name}"; then

				printf "Warning: There was a problem setting up the user with usermod, trying manual addition\n"

				printf "%%s:x:%%s:%%s:%%s:%%s:%%s\n" \
					"${container_user_name}" "${container_user_uid}" \
					"${container_user_gid}" "${container_user_name}" \
					"${container_user_home}" "${SHELL:-"/bin/bash"}" >> /etc/passwd
				printf "%%s::1::::::" "${container_user_name}" >> /etc/shadow
			fi
		fi

		printf "sdbx: Create directories and chown them...\n"
		mkdir -p /home/${container_user_name}/.config
		chown -R ${container_user_uid}:${container_user_gid} /home/${container_user_name}
	`)
}

func ReChown() string {
	return TrimDoc(`
		printf "sdbx: Chown mounted folders in HOME owned by (pseudo) root...\n"

		# Helper function to check and chown a single directory
		check_and_chown() {
		  dir_to_check="$1"

		  if [ -d "$dir_to_check" ]; then
		    owner=$(stat -c %U "$dir_to_check" 2>/dev/null)
		    if [ "$owner" = "root" ]; then
		      echo "Changing ownership of $dir_to_check to $container_user_uid:$container_user_gid"
		      sudo chown "$container_user_uid:$container_user_gid" "$dir_to_check"
		    fi
		  fi
		}

		OIFS="$IFS"
		set -f # Disable globbing

		IFS=":"
		for path in $MOUNTED_PATHS; do
		  IFS="$OIFS" # Restore IFS inside the outer loop
		  case "$path" in
		    "$HOME"|"$HOME/"*)
		      current_dir="$HOME"
		      # check_and_chown "$current_dir"
		      remainder="${path#"$HOME"}"
		      remainder="${remainder#/}" # Remove the leading slash if it exists

		      # If there are subdirectories, iterate over them
		      if [ -n "$remainder" ]; then
		        IFS="/"
		        for comp in $remainder; do
		          IFS="$OIFS" # Restore inside loop

		          if [ -n "$comp" ]; then
		            current_dir="$current_dir/$comp"
		            check_and_chown "$current_dir"
		          fi

		          IFS="/" # Set back to slash for next component
		        done
		        IFS="$OIFS"
		      fi
		      ;;
		    *)
		      # Ignore paths not starting with $HOME
		      ;;
		  esac

		  IFS=":" # Set back to colon for the next mounted path
		done

		IFS="$OIFS"
		set +f
	`)
}
