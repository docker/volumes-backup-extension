import { Circle } from "@mui/icons-material";
import { Grid, Typography } from "@mui/material";
import { useContext } from "react";
import { VolumeIcon } from "./VolumeIcon";
import { VolumeInput } from "./VolumeInput";
import { MyContext } from "..";
import { IVolumeRow } from "../hooks/useGetVolumes";

interface Props {
  value?: string;
  hasError?: boolean;
  setHasError?(v: boolean): void;
  onChange?(v: string): void;
  volumes?: IVolumeRow[];
}

export const VolumeOrInput = ({
  value,
  hasError,
  setHasError,
  onChange,
  volumes,
}: Props) => {
  const context = useContext(MyContext);
  const selectedVolumeName = context.store.volume?.volumeName;
  const containersUsingVolume =
    context.store.volume?.volumeContainers?.length || 0;

  return (
    <Grid container gap={2}>
      <Grid item pt={1}>
        <VolumeIcon />
      </Grid>
      <Grid item flex={1}>
        {selectedVolumeName ? (
          <>
            <Typography p={1} variant="body1">
              {selectedVolumeName}
            </Typography>
            <Grid container alignItems="center" gap={1}>
              <Circle sx={{ fontSize: "10px" }} />
              <Typography variant="body2">
                in use by {`${containersUsingVolume}`} container
                {containersUsingVolume === 1 ? "" : "s"}
              </Typography>
            </Grid>
          </>
        ) : (
          <VolumeInput
            value={value}
            hasError={hasError}
            setHasError={setHasError}
            onChange={onChange}
            volumes={volumes}
          />
        )}
      </Grid>
    </Grid>
  );
};
