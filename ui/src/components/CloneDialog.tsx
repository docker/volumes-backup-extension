import React, { useContext } from "react";
import {
  Button,
  Typography,
  Grid,
  FormLabel,
  FormControl,
} from "@mui/material";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import { createDockerDesktopClient } from "@docker/extension-api-client";

import { MyContext } from "../index";
import { useNotificationContext } from "../NotificationContext";
import { track } from "../common/track";
import { IVolumeRow } from "../hooks/useGetVolumes";
import { VolumeOrInput } from "./VolumeOrInput";
import { VolumeIcon } from "./VolumeIcon";
import { VolumeInput } from "./VolumeInput";

const ddClient = createDockerDesktopClient();

interface Props {
  open: boolean;
  onClose(): void;
  onCompletion(clonedVolumeName: string, v?: boolean): void;
  volumes: IVolumeRow[];
}

export default function CloneDialog({ ...props }: Props) {
  const context = useContext(MyContext);
  const { sendNotification } = useNotificationContext();
  const [hasError, setHasError] = React.useState<boolean>(false);
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
        props.onCompletion(volumeName, true);
      })
      .catch((error) => {
        sendNotification.error(
          `Failed to clone volume ${context.store.volume.volumeName} to destination volume ${volumeName}: ${error.stderr} Exit code: ${error.code}`
        );
        props.onCompletion(volumeName, false);
      });

    props.onClose();
  };

  return (
    <Dialog fullWidth maxWidth="sm" open={props.open} onClose={props.onClose}>
      <DialogTitle>Clone a volume</DialogTitle>
      <DialogContent>
        <FormControl>
          <FormLabel id="from-label" focused={false}>
            <Typography variant="h3" my={1}>
              From:
            </Typography>
          </FormLabel>
          <VolumeOrInput />
        </FormControl>
        <Grid container direction="column" spacing={2}>
          <Grid item mt={2} sx={{ width: "100%" }}>
            <FormControl>
              <FormLabel id="to-label" focused={false}>
                <Typography variant="h3" mt={3} mb={1}>
                  To:
                </Typography>
              </FormLabel>

              <Grid container gap={2}>
                <Grid item pt={1}>
                  <VolumeIcon />
                </Grid>
                <Grid item flex={1}>
                  <VolumeInput
                    volumes={props.volumes}
                    value={volumeName}
                    onChange={setVolumeName}
                    hasError={hasError}
                    setHasError={setHasError}
                  />
                </Grid>
              </Grid>
            </FormControl>
          </Grid>

          {volumeName !== "" && (
            <Grid item>
              <Typography variant="body1" color="text.secondary">
                The volume will be cloned to a new volume named{" "}
                <strong>{volumeName}</strong>.
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
            props.onClose();
          }}
        >
          Cancel
        </Button>
        <Button
          variant="contained"
          onClick={cloneVolume}
          disabled={volumeName === "" || hasError}
        >
          Clone
        </Button>
      </DialogActions>
    </Dialog>
  );
}
