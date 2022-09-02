import React, { useContext } from "react";
import { Button, TextField, Typography, Grid } from "@mui/material";
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

export default function CloneDialog({ ...props }: Props) {
  const context = useContext(MyContext);
  const { sendNotification } = useNotificationContext();

  const [volumeName, setVolumeName] = React.useState<string>(
    `${context.store.volume.volumeName}-cloned`
  );

  const cloneVolume = () => {
    track({ action: "CloneVolume" });

    ddClient.extension.vm.service
      .post(
        `/volumes/${context.store.volume.volumeName}/clone?destVolume=${volumeName}`,
        {}
      )
      .then(() => {
        sendNotification.info(
          `Volume ${context.store.volume.volumeName} cloned to destination volume ${volumeName}`,
          [
            {
              name: "See volume",
              onClick: () => ddClient.desktopUI.navigate.viewVolume(volumeName),
            },
          ]
        );
      })
      .catch((error) => {
        sendNotification.error(
          `Failed to clone volume ${context.store.volume.volumeName} to destination volume ${volumeName}: ${error.stderr} Exit code: ${error.code}`
        );
      });
    props.onClose(true);
  };

  return (
    <Dialog fullWidth maxWidth="sm" open={props.open} onClose={props.onClose}>
      <DialogTitle>Clone a volume</DialogTitle>
      <DialogContent>
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
        <Button
          variant="outlined"
          onClick={() => {
            track({ action: "CloneVolumeCancel" });
            props.onClose(false);
          }}
        >
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
