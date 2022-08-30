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

const ddClient = createDockerDesktopClient();

interface Props {
  open: boolean;
  onClose(v?: boolean): void;
}

export default function DeleteForeverDialog({ ...props }: Props) {
  const context = useContext(MyContext);
  const { sendNotification } = useNotificationContext();

  const deleteVolume = () => {
    track({ action: "DeleteVolume" });
    ddClient.extension.vm.service
      .post(`/volumes/${context.store.volume.volumeName}/delete`, {})
      .then(() => {
        sendNotification.info(
          `Volume ${context.store.volume.volumeName} deleted`
        );
      })
      .catch((error) => {
        sendNotification.error(
          `Failed to delete volume ${context.store.volume.volumeName}: ${error.stderr} Exit code: ${error.code}`
        );
      });
    props.onClose(true);
  };

  return (
    <Dialog open={props.open} onClose={props.onClose}>
      <DialogTitle>Delete a volume permanently</DialogTitle>
      <DialogContent>
        <DialogContentText>
          The volume will be deleted permanently. This action cannot be undone.
          Are you sure?
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button
          variant="outlined"
          onClick={() => {
            track({ action: "DeleteVolumeCancel" });
            props.onClose(false);
          }}
        >
          Cancel
        </Button>
        <Button variant="contained" onClick={deleteVolume}>
          Delete forever
        </Button>
      </DialogActions>
    </Dialog>
  );
}
