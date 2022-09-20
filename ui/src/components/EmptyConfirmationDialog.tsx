import { useContext } from "react";
import { Button } from "@mui/material";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogContentText from "@mui/material/DialogContentText";
import DialogTitle from "@mui/material/DialogTitle";
import { createDockerDesktopClient } from "@docker/extension-api-client";

import { MyContext } from "../index";
import { useNotificationContext } from "../NotificationContext";
import { track } from "../common/track";
import { isError } from "../common/isError";

const ddClient = createDockerDesktopClient();

interface Props {
  open: boolean;
  onClose(): void;
  onCompletion(v?: boolean): void;
}

export default function EmptyConfirmationDialog({ ...props }: Props) {
  const context = useContext(MyContext);
  const { sendNotification } = useNotificationContext();

  const emptyVolume = () => {
    track({ action: "EmptyVolume" });
    ddClient.docker.cli
      .exec("run", [
        "--rm",
        "--label com.volumes-backup-extension.trigger-ui-refresh=true",
        "--label com.docker.compose.project=docker_volumes-backup-extension-desktop-extension",
        `-v=${context.store.volume.volumeName}:/vackup-volume `,
        "busybox",
        "/bin/sh",
        "-c",
        '"rm -rf /vackup-volume/..?* /vackup-volume/.[!.]* /vackup-volume/*"', // hidden and not-hidden files and folders: .[!.]* matches all dot files except . and files whose name begins with .., and ..?* matches all dot-dot files except ..
      ])
      .then((output) => {
        if (isError(output.stderr)) {
          sendNotification.error(output.stderr);
          props.onCompletion(false);
          return;
        }
        sendNotification.info(
          `The content of volume ${context.store.volume.volumeName} has been removed`
        );
        props.onCompletion(true);
      })
      .catch((error) => {
        sendNotification.error(
          `Failed to empty volume ${context.store.volume.volumeName}: ${error.stderr} Exit code: ${error.code}`
        );
        props.onCompletion(false);
      });
    props.onClose();
  };

  return (
    <Dialog open={props.open} onClose={props.onClose}>
      <DialogTitle>Empty a volume</DialogTitle>
      <DialogContent>
        <DialogContentText>
          The volume <strong>{context.store.volume.volumeName}</strong> will be
          emptied. This action cannot be undone. Are you sure?
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button
          variant="outlined"
          onClick={() => {
            track({ action: "DeleteVolumeCancel" });
            props.onClose();
          }}
        >
          Cancel
        </Button>
        <Button variant="contained" onClick={emptyVolume}>
          Empty
        </Button>
      </DialogActions>
    </Dialog>
  );
}
