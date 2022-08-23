import React, { useContext } from "react";
import {
  Button,
  TextField,
  Typography,
  Grid,
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
import { isError } from "../common/isError";
import { useNotificationContext } from "../NotificationContext";

const client = createDockerDesktopClient();

function useDockerDesktopClient() {
  return client;
}

export default function CloneDialog({ ...props }) {
  const ddClient = useDockerDesktopClient();
  const context = useContext(MyContext);
  const { sendNotification } = useNotificationContext();

  const [volumeName, setVolumeName] = React.useState<string>(
    `${context.store.volume.volumeName}-cloned`
  );
  const [actionInProgress, setActionInProgress] =
    React.useState<boolean>(false);

  const cloneVolume = async () => {
    setActionInProgress(true);
    let actionSuccessfullyCompleted = false;

    try {
      // TODO: check if destination volume already exists
      const createVolumeOutput = await ddClient.docker.cli.exec("volume", [
        "create",
        volumeName,
      ]);
      if (createVolumeOutput.stderr !== "") {
        sendNotification.error(createVolumeOutput.stderr);
        return;
      }

      const cloneOutput = await ddClient.docker.cli.exec("run", [
        "--rm",
        `-v=${context.store.volume.volumeName}:/from`,
        `-v=${volumeName}:/to`,
        "alpine",
        "ash",
        "-c",
        '"cd /from ; cp -av . /to"',
      ]);
      if (isError(cloneOutput.stderr)) {
        sendNotification.error(cloneOutput.stderr);
        return;
      }

      sendNotification.info(
        `Volume ${context.store.volume.volumeName} cloned to destination volume ${volumeName}`,
        [
          {
            name: "See volume",
            onClick: () =>
              ddClient.desktopUI.navigate.viewVolume(
                context.store.volume.volumeName
              ),
          },
        ]
      );

      actionSuccessfullyCompleted = true;
    } catch (error) {
      sendNotification.error(
        `Failed to clone volume ${context.store.volume.volumeName} to destinaton volume ${volumeName}: ${error.stderr} Exit code: ${error.code}`
      );
    } finally {
      setActionInProgress(false);
      props.onClose(actionSuccessfullyCompleted);
    }
  };

  return (
    <Dialog fullWidth maxWidth="sm" open={props.open} onClose={props.onClose}>
      <DialogTitle>Clone a volume</DialogTitle>
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
        <DialogContentText>Clones a volume.</DialogContentText>

        <Grid container direction="column" spacing={2}>
          <Grid item mt={2}>
            <TextField
              required
              autoFocus
              id="volume-name"
              label="Volume name"
              fullWidth
              defaultValue={`${context.store.volume.volumeName}-cloned`}
              spellCheck={false}
              onChange={(e) => {
                setVolumeName(e.target.value);
              }}
            />
          </Grid>

          {volumeName !== "" && (
            <Grid item>
              <Typography variant="body1" color="text.secondary">
                The volume will be cloned to a new volume named {volumeName}.
              </Typography>
            </Grid>
          )}
        </Grid>
      </DialogContent>
      <DialogActions>
        <Button variant="outlined" onClick={() => props.onClose(false)}>
          Cancel
        </Button>
        <Button
          variant="contained"
          onClick={cloneVolume}
          disabled={volumeName === ""}
        >
          Clone
        </Button>
      </DialogActions>
    </Dialog>
  );
}
