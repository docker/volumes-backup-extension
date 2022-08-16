import React, { useContext } from "react";
import {
  Button,
  Backdrop,
  CircularProgress,
} from "@mui/material";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogContentText from "@mui/material/DialogContentText";
import DialogTitle from "@mui/material/DialogTitle";
import { createDockerDesktopClient } from "@docker/extension-api-client";

import { MyContext } from "../index";
import { useNotificationContext } from "../NotificationContext";

const client = createDockerDesktopClient();

function useDockerDesktopClient() {
  return client;
}

export default function DeleteForeverDialog({ ...props }) {
  const ddClient = useDockerDesktopClient();
  const context = useContext(MyContext);
  const { sendNotification } = useNotificationContext();

  const [actionInProgress, setActionInProgress] =
    React.useState<boolean>(false);

  const deleteVolume = async () => {
    setActionInProgress(true);
    let actionSuccessfullyCompleted = false

    try {
      // TODO: check if volume already exists
      const output = await ddClient.docker.cli.exec("volume", [
        "rm",
        context.store.volume.volumeName,
      ]);
      if (output.stderr !== "") {
        sendNotification(output.stderr);
        return;
      }

      sendNotification(
        `Volume ${context.store.volume.volumeName} deleted`
      );

      actionSuccessfullyCompleted = true
    } catch (error) {
      sendNotification(
        `Failed to delete volume ${context.store.volume.volumeName}: ${error.stderr} Exit code: ${error.code}`
      );
    } finally {
      setActionInProgress(false);
      props.onClose(actionSuccessfullyCompleted)
    }
  };

  return (
    <Dialog open={props.open} onClose={props.onClose}>
      <DialogTitle>Delete a volume permanently</DialogTitle>
      <DialogContent>
        <Backdrop
          sx={{
            backgroundColor: "rgba(245,244,244,0.4)",
            zIndex: (theme) => theme.zIndex.drawer + 1,
          }}
          open={actionInProgress}
        >
          <CircularProgress color="info" />
        </Backdrop>
        <DialogContentText>The volume will be deleted permanently. This action cannot be undone. Are you sure?</DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button variant="outlined" onClick={() => props.onClose(false)}>Cancel</Button>
        <Button variant="contained" onClick={deleteVolume}>
          Delete forever
        </Button>
      </DialogActions>
    </Dialog>
  );
}
